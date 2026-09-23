package multiprocess

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/metrics"
	"github.com/yyyar/gobetween/stats"
)

const heartbeatTimeout = 15 * time.Second

var supervisorLog = logging.For("supervisor")

type workerSlot struct {
	id            int
	cpus          []int
	generation    uint64
	restarts      uint64
	command       *exec.Cmd
	channel       *Channel
	ready         bool
	startedAt     time.Time
	lastHeartbeat time.Time
	snapshot      map[string]stats.Stats
}

type workerEvent struct {
	workerID   int
	generation uint64
	kind       string
	message    Message
	err        error
}

const (
	eventMessage = "message"
	eventReadEnd = "read-end"
	eventExit    = "exit"
	eventRestart = "restart"
)

// RunSupervisor starts CPU-pinned UDP workers, aggregates their metrics, and
// keeps them alive until SIGINT or SIGTERM.
func RunSupervisor(cfg config.Config) error {
	if !platformSupported() {
		return fmt.Errorf("UDP multiprocess runtime requires Linux")
	}
	if err := validateSupervisorConfig(cfg); err != nil {
		return err
	}

	available, err := allowedCPUs()
	if err != nil {
		return fmt.Errorf("read allowed CPU set: %w", err)
	}
	partitions, err := splitCPUs(available, cfg.Runtime.WorkerProcesses)
	if err != nil {
		return err
	}
	if cfg.Runtime.WorkerProcesses > len(available) {
		supervisorLog.Warnf("worker_processes=%d exceeds available CPUs=%d; workers will share CPUs", cfg.Runtime.WorkerProcesses, len(available))
	}

	restartEnabled := true
	if cfg.Runtime.RestartWorkers != nil {
		restartEnabled = *cfg.Runtime.RestartWorkers
	}
	restartBackoff, err := configuredDuration(cfg.Runtime.RestartBackoff, time.Second, "restart_backoff")
	if err != nil {
		return err
	}
	shutdownTimeout, err := configuredDuration(cfg.Runtime.ShutdownTimeout, 10*time.Second, "shutdown_timeout")
	if err != nil {
		return err
	}

	metrics.Start(cfg.Metrics)
	events := make(chan workerEvent, cfg.Runtime.WorkerProcesses*4)
	slots := make([]*workerSlot, cfg.Runtime.WorkerProcesses)
	for index := range slots {
		slots[index] = &workerSlot{id: index + 1, cpus: partitions[index]}
		metrics.ReportWorkerState(index+1, false, 0)
		if err := startWorker(slots[index], events); err != nil {
			shutdownWorkerSet(slots, events, shutdownTimeout)
			return fmt.Errorf("start worker %d: %w", index+1, err)
		}
	}

	supervisorLog.Infof("Started %d UDP workers on allowed CPUs %s", len(slots), formatCPUList(available))
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	watchdog := time.NewTicker(2 * time.Second)
	defer watchdog.Stop()

	for {
		select {
		case received := <-signals:
			supervisorLog.Infof("Received %s; stopping UDP workers", received)
			shutdownWorkerSet(slots, events, shutdownTimeout)
			metrics.ReplaceSnapshots(map[string]metrics.ServerSnapshot{})
			return nil
		case event := <-events:
			handleWorkerEvent(event, slots, events, restartEnabled, restartBackoff)
		case now := <-watchdog.C:
			for _, slot := range slots {
				if slot.command == nil {
					continue
				}
				lastSeen := slot.lastHeartbeat
				if lastSeen.IsZero() {
					lastSeen = slot.startedAt
				}
				if now.Sub(lastSeen) > heartbeatTimeout {
					supervisorLog.Errorf("Worker %d heartbeat timed out; terminating pid=%d", slot.id, slot.command.Process.Pid)
					_ = slot.command.Process.Kill()
				}
			}
		}
	}
}

func validateSupervisorConfig(cfg config.Config) error {
	if cfg.Runtime.WorkerProcesses <= 0 {
		return fmt.Errorf("worker_processes must be greater than zero")
	}
	if cfg.Api.Enabled {
		return fmt.Errorf("REST API is not supported by the UDP multiprocess runtime")
	}
	if cfg.Profiler != nil && cfg.Profiler.Enabled {
		return fmt.Errorf("profiler is not supported by the UDP multiprocess runtime")
	}
	return validateWorkerConfig(cfg)
}

func configuredDuration(value string, fallback time.Duration, field string) (time.Duration, error) {
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid runtime.%s %q", field, value)
	}
	return duration, nil
}

func startWorker(slot *workerSlot, events chan<- workerEvent) error {
	parentFile, childFile, err := socketPair()
	if err != nil {
		return err
	}

	connection, err := net.FileConn(parentFile)
	_ = parentFile.Close()
	if err != nil {
		_ = childFile.Close()
		return err
	}
	channel := newChannel(connection)

	executable, err := os.Executable()
	if err != nil {
		_ = channel.Close()
		_ = childFile.Close()
		return err
	}
	command := exec.Command(executable, os.Args[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.ExtraFiles = []*os.File{childFile}
	configureWorkerCommand(command)

	env := replaceEnv(os.Environ(), "GOBETWEEN", "")
	env = replaceEnv(env, envRole, roleBootstrap)
	env = replaceEnv(env, envWorkerID, strconv.Itoa(slot.id))
	env = replaceEnv(env, envWorkerCPUs, formatCPUList(slot.cpus))
	env = replaceEnv(env, envIPCFile, "3")
	env = replaceEnv(env, envReusePort, "1")
	env = replaceEnv(env, "GOMAXPROCS", "")
	command.Env = env

	if err := command.Start(); err != nil {
		_ = channel.Close()
		_ = childFile.Close()
		return err
	}
	_ = childFile.Close()

	slot.generation++
	slot.command = command
	slot.channel = channel
	slot.ready = false
	slot.startedAt = time.Now()
	slot.lastHeartbeat = time.Time{}
	slot.snapshot = nil
	generation := slot.generation
	workerID := slot.id

	supervisorLog.Infof("Started worker %d pid=%d cpus=%s GOMAXPROCS=%d", workerID, command.Process.Pid, formatCPUList(slot.cpus), len(slot.cpus))
	go readWorker(workerID, generation, channel, events)
	go func() {
		waitErr := command.Wait()
		events <- workerEvent{workerID: workerID, generation: generation, kind: eventExit, err: waitErr}
	}()
	return nil
}

func readWorker(workerID int, generation uint64, channel *Channel, events chan<- workerEvent) {
	for {
		message, err := channel.Receive()
		if err != nil {
			events <- workerEvent{workerID: workerID, generation: generation, kind: eventReadEnd, err: err}
			return
		}
		events <- workerEvent{workerID: workerID, generation: generation, kind: eventMessage, message: message}
	}
}

func handleWorkerEvent(event workerEvent, slots []*workerSlot, events chan<- workerEvent, restartEnabled bool, restartBackoff time.Duration) {
	if event.workerID <= 0 || event.workerID > len(slots) {
		return
	}
	slot := slots[event.workerID-1]
	if event.generation != slot.generation {
		return
	}

	switch event.kind {
	case eventMessage:
		switch event.message.Type {
		case messageReady:
			slot.ready = true
			slot.lastHeartbeat = time.Now()
			metrics.ReportWorkerState(slot.id, true, slot.restarts)
			supervisorLog.Infof("Worker %d ready pid=%d", slot.id, event.message.PID)
		case messageHeartbeat:
			slot.lastHeartbeat = time.Now()
			metrics.ReportWorkerState(slot.id, true, slot.restarts)
		case messageStats:
			slot.lastHeartbeat = time.Now()
			slot.snapshot = event.message.Stats
			publishAggregate(slots)
		case messageError:
			supervisorLog.Errorf("Worker %d error: %s", slot.id, event.message.Error)
		}
	case eventReadEnd:
		// Wait owns lifecycle and restart decisions; the socket normally closes
		// just before the process exit notification arrives.
	case eventExit:
		if slot.channel != nil {
			_ = slot.channel.Close()
		}
		slot.command = nil
		slot.channel = nil
		slot.ready = false
		slot.snapshot = nil
		metrics.ReportWorkerState(slot.id, false, slot.restarts)
		publishAggregate(slots)
		if event.err != nil {
			supervisorLog.Errorf("Worker %d exited: %v", slot.id, event.err)
		} else {
			supervisorLog.Warnf("Worker %d exited", slot.id)
		}
		if restartEnabled {
			slot.restarts++
			metrics.ReportWorkerState(slot.id, false, slot.restarts)
			generation := slot.generation
			go func(workerID int) {
				timer := time.NewTimer(restartBackoff)
				defer timer.Stop()
				<-timer.C
				events <- workerEvent{workerID: workerID, generation: generation, kind: eventRestart}
			}(slot.id)
		}
	case eventRestart:
		if slot.command != nil {
			return
		}
		if err := startWorker(slot, events); err != nil {
			supervisorLog.Errorf("Failed to restart worker %d: %v", slot.id, err)
			generation := slot.generation
			go func(workerID int) {
				timer := time.NewTimer(restartBackoff)
				defer timer.Stop()
				<-timer.C
				events <- workerEvent{workerID: workerID, generation: generation, kind: eventRestart}
			}(slot.id)
		}
	}
}

func shutdownWorkerSet(slots []*workerSlot, events <-chan workerEvent, timeout time.Duration) {
	remaining := 0
	for _, slot := range slots {
		if slot.command == nil {
			continue
		}
		remaining++
		if slot.channel != nil {
			_ = slot.channel.Send(Message{Type: messageShutdown, WorkerID: slot.id, SentAt: time.Now().UnixNano()})
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for remaining > 0 {
		select {
		case event := <-events:
			if event.kind != eventExit || event.workerID <= 0 || event.workerID > len(slots) {
				continue
			}
			slot := slots[event.workerID-1]
			if event.generation != slot.generation || slot.command == nil {
				continue
			}
			if slot.channel != nil {
				_ = slot.channel.Close()
			}
			slot.command = nil
			slot.channel = nil
			remaining--
		case <-timer.C:
			for _, slot := range slots {
				if slot.command != nil {
					supervisorLog.Warnf("Force killing worker %d pid=%d after shutdown timeout", slot.id, slot.command.Process.Pid)
					_ = slot.command.Process.Kill()
				}
			}
			return
		}
	}
}

func publishAggregate(slots []*workerSlot) {
	metrics.ReplaceSnapshots(aggregateSnapshots(slots))
}

func aggregateSnapshots(slots []*workerSlot) map[string]metrics.ServerSnapshot {
	servers := make(map[string]*metrics.ServerSnapshot)
	backendMaps := make(map[string]map[core.Target]*core.Backend)
	for _, slot := range slots {
		for name, snapshot := range slot.snapshot {
			aggregate := servers[name]
			if aggregate == nil {
				aggregate = &metrics.ServerSnapshot{}
				servers[name] = aggregate
				backendMaps[name] = make(map[core.Target]*core.Backend)
			}
			aggregate.ActiveConnections += snapshot.ActiveConnections
			aggregate.RxTotal += snapshot.RxTotal
			aggregate.TxTotal += snapshot.TxTotal
			aggregate.RxSecond += snapshot.RxSecond
			aggregate.TxSecond += snapshot.TxSecond
			for _, backend := range snapshot.Backends {
				current := backendMaps[name][backend.Target]
				if current == nil {
					copy := backend
					backendMaps[name][backend.Target] = &copy
					continue
				}
				current.Stats.Live = current.Stats.Live || backend.Stats.Live
				current.Stats.Discovered = current.Stats.Discovered || backend.Stats.Discovered
				current.Stats.TotalConnections += backend.Stats.TotalConnections
				current.Stats.ActiveConnections += backend.Stats.ActiveConnections
				current.Stats.RefusedConnections += backend.Stats.RefusedConnections
				current.Stats.RxBytes += backend.Stats.RxBytes
				current.Stats.TxBytes += backend.Stats.TxBytes
				current.Stats.RxSecond += backend.Stats.RxSecond
				current.Stats.TxSecond += backend.Stats.TxSecond
			}
		}
	}

	result := make(map[string]metrics.ServerSnapshot, len(servers))
	for name, aggregate := range servers {
		for _, backend := range backendMaps[name] {
			aggregate.Backends = append(aggregate.Backends, *backend)
		}
		sort.Slice(aggregate.Backends, func(i, j int) bool {
			left := aggregate.Backends[i].Host + ":" + aggregate.Backends[i].Port
			right := aggregate.Backends[j].Host + ":" + aggregate.Backends[j].Port
			return left < right
		})
		result[name] = *aggregate
	}
	return result
}
