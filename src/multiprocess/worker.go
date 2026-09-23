package multiprocess

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/manager"
	"github.com/yyyar/gobetween/metrics"
	"github.com/yyyar/gobetween/stats"
)

const (
	heartbeatInterval = 2 * time.Second
	statsInterval     = 2 * time.Second
)

// RunWorker starts all configured UDP servers in a child process and reports
// health and statistics to its supervisor.
func RunWorker(cfg config.Config) error {
	workerID, err := strconv.Atoi(os.Getenv(envWorkerID))
	if err != nil || workerID <= 0 {
		return fmt.Errorf("invalid worker id %q", os.Getenv(envWorkerID))
	}

	channel, err := inheritedChannel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := validateWorkerConfig(cfg); err != nil {
		_ = channel.Send(Message{Type: messageError, WorkerID: workerID, PID: os.Getpid(), Error: err.Error()})
		return err
	}

	// Workers never bind HTTP endpoints. Starting metrics in disabled mode also
	// makes the existing server instrumentation callbacks safe no-ops.
	cfg.Api.Enabled = false
	cfg.Metrics.Enabled = false
	cfg.Profiler = nil
	cfg.Runtime.WorkerProcesses = 0
	metrics.Start(cfg.Metrics)
	manager.Initialize(cfg)

	if err := channel.Send(Message{Type: messageReady, WorkerID: workerID, PID: os.Getpid(), SentAt: time.Now().UnixNano()}); err != nil {
		manager.StopAll()
		return err
	}

	messages := make(chan Message, 1)
	readErrors := make(chan error, 1)
	go func() {
		for {
			message, receiveErr := channel.Receive()
			if receiveErr != nil {
				readErrors <- receiveErr
				return
			}
			messages <- message
		}
	}()

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	statsTicker := time.NewTicker(statsInterval)
	defer heartbeatTicker.Stop()
	defer statsTicker.Stop()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	shutdown := func() error {
		manager.StopAll()
		return channel.Send(Message{Type: messageStopped, WorkerID: workerID, PID: os.Getpid(), SentAt: time.Now().UnixNano()})
	}

	for {
		select {
		case message := <-messages:
			if message.Type == messageShutdown {
				return shutdown()
			}
		case err := <-readErrors:
			manager.StopAll()
			return fmt.Errorf("supervisor IPC closed: %w", err)
		case <-signals:
			return shutdown()
		case <-heartbeatTicker.C:
			if err := channel.Send(Message{Type: messageHeartbeat, WorkerID: workerID, PID: os.Getpid(), SentAt: time.Now().UnixNano()}); err != nil {
				manager.StopAll()
				return err
			}
		case <-statsTicker.C:
			if err := channel.Send(Message{Type: messageStats, WorkerID: workerID, PID: os.Getpid(), SentAt: time.Now().UnixNano(), Stats: stats.SnapshotAll()}); err != nil {
				manager.StopAll()
				return err
			}
		}
	}
}

func inheritedChannel() (*Channel, error) {
	fd, err := strconv.Atoi(os.Getenv(envIPCFile))
	if err != nil || fd < 3 {
		return nil, fmt.Errorf("invalid inherited IPC descriptor %q", os.Getenv(envIPCFile))
	}
	file := os.NewFile(uintptr(fd), "gobetween-supervisor-ipc")
	if file == nil {
		return nil, fmt.Errorf("open inherited IPC descriptor %d", fd)
	}
	connection, err := net.FileConn(file)
	_ = file.Close()
	if err != nil {
		return nil, fmt.Errorf("open supervisor IPC channel: %w", err)
	}
	return newChannel(connection), nil
}

func validateWorkerConfig(cfg config.Config) error {
	if len(cfg.Servers) == 0 {
		return fmt.Errorf("multiprocess runtime requires at least one UDP server")
	}
	for name, server := range cfg.Servers {
		if server.Protocol != "udp" {
			return fmt.Errorf("server %q uses protocol %q; multiprocess runtime supports udp only", name, server.Protocol)
		}
	}
	return nil
}
