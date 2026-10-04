//go:build linux

package test

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Opt-in process integration tests: build gobetween first, then set
// GOBETWEEN_TEST_BINARY=/absolute/path/gobetween. RR needs BPF privileges.
type proxyLog struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (l *proxyLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffer.Write(p)
}
func (l *proxyLog) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.buffer.String() }

var readyWorkerPattern = regexp.MustCompile(`Worker (\d+) ready pid=(\d+)`)

var startedCPUWorkerPattern = regexp.MustCompile(`Started worker (\d+) pid=(\d+) cpus=([0-9,]+) GOMAXPROCS=(\d+)`)

// Check the real child threads, not just the parent's planned partition.
func verifyPhysicalWorkerCPUs(t *testing.T, log *proxyLog, workers int) map[int]string {
	t.Helper()
	pids := readyPIDs(log)
	coreOwners := make(map[string]int)
	masks := make(map[int]string)
	for worker := 1; worker <= workers; worker++ {
		pid := pids[worker]
		var mask *big.Int
		for _, match := range startedCPUWorkerPattern.FindAllStringSubmatch(log.String(), -1) {
			loggedPID, _ := strconv.Atoi(match[2])
			if loggedPID != pid {
				continue
			}
			mask = new(big.Int)
			cpus := strings.Split(match[3], ",")
			for _, value := range cpus {
				cpu, err := strconv.Atoi(value)
				if err != nil {
					t.Fatal(err)
				}
				mask.SetBit(mask, cpu, 1)
			}
			procs, _ := strconv.Atoi(match[4])
			if procs != len(cpus) {
				t.Fatalf("worker %d GOMAXPROCS=%d, CPUs=%v", worker, procs, cpus)
			}
		}
		if mask == nil {
			t.Fatalf("missing CPU allocation log for worker %d pid=%d", worker, pid)
		}
		masks[worker] = mask.Text(16)
		for cpu := 0; cpu < mask.BitLen(); cpu++ {
			if mask.Bit(cpu) == 0 {
				continue
			}
			path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/topology/core_cpus", cpu)
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				data, err = os.ReadFile(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/topology/thread_siblings", cpu))
			}
			if err != nil {
				t.Fatal(err)
			}
			core, ok := new(big.Int).SetString(strings.ReplaceAll(strings.TrimSpace(string(data)), ",", ""), 16)
			if !ok || core.Sign() == 0 {
				t.Fatalf("invalid core mask for CPU %d: %q", cpu, data)
			}
			if new(big.Int).And(new(big.Int).Set(core), mask).Cmp(core) != 0 {
				t.Fatalf("worker %d has incomplete core for CPU %d", worker, cpu)
			}
			key := core.Text(16)
			if owner, exists := coreOwners[key]; exists && owner != worker {
				t.Fatalf("workers %d and %d share physical core %s", owner, worker, key)
			}
			coreOwners[key] = worker
		}
		paths, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/status", pid))
		if err != nil || len(paths) == 0 {
			t.Fatalf("cannot inspect worker %d threads: %v", worker, err)
		}
		checked := 0
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue
			} // A runtime thread can exit during inspection.
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, line := range strings.Split(string(data), "\n") {
				if !strings.HasPrefix(line, "Cpus_allowed:") {
					continue
				}
				actual, ok := new(big.Int).SetString(strings.ReplaceAll(strings.TrimSpace(strings.TrimPrefix(line, "Cpus_allowed:")), ",", ""), 16)
				if !ok || actual.Cmp(mask) != 0 {
					t.Fatalf("worker %d thread affinity differs from planned mask: %s", worker, line)
				}
				found = true
			}
			if !found {
				t.Fatalf("thread CPU mask missing: %s", path)
			}
			checked++
		}
		if checked == 0 {
			t.Fatalf("no worker %d threads inspected", worker)
		}
	}
	return masks
}

func readyPIDs(log *proxyLog) map[int]int {
	result := make(map[int]int)
	for _, match := range readyWorkerPattern.FindAllStringSubmatch(log.String(), -1) {
		id, _ := strconv.Atoi(match[1])
		pid, _ := strconv.Atoi(match[2])
		result[id] = pid
	}
	return result
}

func eventually(t *testing.T, description string, test func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if test() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out: %s", description)
}

func reserveUDP(t *testing.T) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := conn.LocalAddr().String()
	_ = conn.Close()
	return address
}

func reserveTCP(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

type backendPacket struct {
	backend int
	peer    string
	payload []byte
}

func startProxy(t *testing.T, binary, configuration string) (*exec.Cmd, *proxyLog) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "proxy.toml")
	if err := os.WriteFile(file, []byte(configuration), 0600); err != nil {
		t.Fatal(err)
	}
	log := &proxyLog{}
	command := exec.Command(binary, "-c", file)
	command.Stdout = log
	command.Stderr = log
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	t.Cleanup(func() {
		_ = command.Process.Signal(syscall.SIGTERM)
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("proxy exited: %v\n%s", err, log.String())
			}
			if strings.Contains(log.String(), "Force killing worker") {
				t.Errorf("worker graceful shutdown timed out\n%s", log.String())
			}
			// The supervisor can exit successfully even when a child panics.
			if strings.Contains(log.String(), "panic:") || strings.Contains(log.String(), "DATA RACE") {
				t.Errorf("worker failed during integration test\n%s", log.String())
			}
		case <-time.After(8 * time.Second):
			_ = command.Process.Kill()
			<-done
			t.Errorf("proxy did not shut down gracefully\n%s", log.String())
		}
	})
	return command, log
}

func TestUDPWorkerIntegration(t *testing.T) {
	binary := os.Getenv("GOBETWEEN_TEST_BINARY")
	if binary == "" {
		t.Skip("set GOBETWEEN_TEST_BINARY to opt in")
	}
	for _, tc := range []struct {
		name, mode, cpuPolicy string
		workers               int
		responses, restart    bool
	}{
		{"rr_one_way_restart", "rr", "logical", 4, false, true},
		{"rr_request_response", "rr", "logical", 4, true, false},
		{"hash_one_way", "hash", "logical", 4, false, false},
		{"rr_single_worker", "rr", "logical", 1, false, false},
		{"physical_rr_one_way_restart", "rr", "physical", 4, false, true},
		{"physical_rr_request_response", "rr", "physical", 4, true, false},
		{"physical_hash_one_way", "hash", "physical", 4, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			packets := make(chan backendPacket, 1024)
			var addresses []string
			for backend := 0; backend < 3; backend++ {
				conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = conn.Close() })
				addresses = append(addresses, strconv.Quote(conn.LocalAddr().String()))
				go func(index int, socket *net.UDPConn) {
					buffer := make([]byte, 65535)
					for {
						n, peer, err := socket.ReadFromUDP(buffer)
						if err != nil {
							return
						}
						payload := append([]byte(nil), buffer[:n]...)
						packets <- backendPacket{index, peer.String(), payload}
						if tc.responses {
							_, _ = socket.WriteToUDP(payload, peer)
						}
					}
				}(backend, conn)
			}
			bind, metrics := reserveUDP(t), reserveTCP(t)
			maxRequests := 1
			if tc.responses {
				maxRequests = 0
			}
			configuration := fmt.Sprintf(`[runtime]
worker_processes = %d
worker_cpu_policy = %q
restart_backoff = "200ms"
shutdown_timeout = "5s"
[logging]
level = "info"
output = "stdout"
[metrics]
enabled = true
bind = %q
[servers.proxy]
bind = %q
protocol = "udp"
balance = "roundrobin"
client_idle_timeout = "30s"
backend_idle_timeout = "30s"
[servers.proxy.udp]
reuse_port_distribution = %q
max_requests = %d
max_responses = 0
[servers.proxy.discovery]
kind = "static"
static_list = [%s]
`, tc.workers, tc.cpuPolicy, metrics, bind, tc.mode, maxRequests, strings.Join(addresses, ","))
			_, log := startProxy(t, binary, configuration)
			eventually(t, "workers ready", func() bool { return len(readyPIDs(log)) == tc.workers })
			var physicalMasks map[int]string
			if tc.cpuPolicy == "physical" {
				physicalMasks = verifyPhysicalWorkerCPUs(t, log, tc.workers)
			}
			// Discovery is asynchronous and predates the RR implementation.
			// Wait for published backend statistics before starting assertions.
			httpClient := &http.Client{Timeout: time.Second}
			eventually(t, "discovery and parent metrics", func() bool {
				response, err := httpClient.Get("http://" + metrics + "/metrics")
				if err != nil {
					return false
				}
				defer response.Body.Close()
				data, _ := io.ReadAll(response.Body)
				return strings.Contains(string(data), `gobetween_backend_live{`) && strings.Count(string(data), `gobetween_worker_up{`) == tc.workers
			})
			address, _ := net.ResolveUDPAddr("udp", bind)
			client, err := net.DialUDP("udp", nil, address)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			send := func(sequence int) backendPacket {
				payload := append([]byte{0, 255, 0, 128}, []byte(fmt.Sprintf("message-%d", sequence))...)
				if _, err := client.Write(payload); err != nil {
					t.Fatal(err)
				}
				var packet backendPacket
				select {
				case packet = <-packets:
				case <-time.After(2 * time.Second):
					t.Fatalf("missing packet %d\n%s", sequence, log.String())
				}
				if !bytes.Equal(packet.payload, payload) {
					t.Fatalf("payload mismatch for %d", sequence)
				}
				if tc.responses {
					_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
					buffer := make([]byte, 65535)
					n, err := client.Read(buffer)
					if err != nil {
						t.Fatalf("response %d: %v", sequence, err)
					}
					if !bytes.Equal(buffer[:n], payload) {
						t.Fatalf("response payload mismatch for %d", sequence)
					}
				}
				return packet
			}
			peers := make(map[string]int)
			backendCounts := make(map[int]int)
			for sequence := 0; sequence < 144; sequence++ {
				packet := send(sequence)
				peers[packet.peer]++
				backendCounts[packet.backend]++
			}
			expectedWorkers := tc.workers
			if tc.mode == "hash" {
				expectedWorkers = 1
			}
			expectedPeers := expectedWorkers
			if !tc.responses {
				expectedPeers *= 3
			}
			if len(peers) != expectedPeers {
				t.Fatalf("backend socket count=%d want=%d (%v)", len(peers), expectedPeers, peers)
			}
			for peer, count := range peers {
				if count != 144/expectedPeers {
					t.Errorf("peer %s received %d want %d", peer, count, 144/expectedPeers)
				}
			}
			if !tc.responses {
				for backend := 0; backend < 3; backend++ {
					if backendCounts[backend] != 48 {
						t.Errorf("backend %d received %d want 48", backend, backendCounts[backend])
					}
				}
			}
			if tc.restart {
				old := readyPIDs(log)
				// Kill all children to exercise an empty group followed by full
				// recovery. No traffic is sent during the intentional outage.
				for _, pid := range old {
					process, err := os.FindProcess(pid)
					if err != nil {
						t.Fatal(err)
					}
					if err := process.Kill(); err != nil {
						t.Fatal(err)
					}
				}
				eventually(t, "all workers restarted", func() bool {
					current := readyPIDs(log)
					for id, pid := range old {
						if current[id] == pid || current[id] == 0 {
							return false
						}
					}
					return true
				})
				time.Sleep(100 * time.Millisecond)
				if tc.cpuPolicy == "physical" {
					currentMasks := verifyPhysicalWorkerCPUs(t, log, tc.workers)
					for worker, previous := range physicalMasks {
						if currentMasks[worker] != previous {
							t.Fatalf("worker %d CPU mask changed after restart", worker)
						}
					}
				}
				newPeers := make(map[string]int)
				for sequence := 144; sequence < 288; sequence++ {
					packet := send(sequence)
					newPeers[packet.peer]++
				}
				if len(newPeers) != 12 {
					t.Fatalf("restarted socket count=%d want 12", len(newPeers))
				}
				for peer, count := range newPeers {
					if count != 12 {
						t.Errorf("restarted peer %s count=%d want 12", peer, count)
					}
				}
			}
		})
	}
}
