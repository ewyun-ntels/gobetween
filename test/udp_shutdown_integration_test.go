//go:build linux

package test

import (
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Keep traffic running through SIGTERM, including while statistics tickers
// are active. startProxy checks child panic/race logs, not just parent status.
func TestUDPShutdownUnderTraffic(t *testing.T) {
	binary := os.Getenv("GOBETWEEN_TEST_BINARY")
	if binary == "" {
		t.Skip("set GOBETWEEN_TEST_BINARY to opt in; rr requires BPF privileges")
	}
	for _, mode := range []string{"hash", "rr"} {
		for _, maxRequests := range []int{1, 0} {
			for iteration := 0; iteration < 3; iteration++ {
				t.Run(fmt.Sprintf("%s/requests_%d/%d", mode, maxRequests, iteration), func(t *testing.T) {
					backend, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
					if err != nil {
						t.Fatal(err)
					}
					var packets atomic.Int64
					backendDone := make(chan struct{})
					t.Cleanup(func() { _ = backend.Close(); <-backendDone })
					go func() {
						defer close(backendDone)
						b := make([]byte, 65535)
						for {
							n, peer, err := backend.ReadFromUDP(b)
							if err != nil {
								return
							}
							packets.Add(1)
							if maxRequests == 0 {
								_, _ = backend.WriteToUDP(b[:n], peer)
							}
						}
					}()
					bind := reserveUDP(t)
					stopped := make(chan struct{})
					var clients sync.WaitGroup
					var client *net.UDPConn
					var replies atomic.Int64
					// Cleanup is LIFO: startProxy stops the proxy before traffic.
					t.Cleanup(func() {
						close(stopped)
						if client != nil {
							_ = client.Close()
						}
						clients.Wait()
					})
					configuration := fmt.Sprintf(`[runtime]
worker_processes = 2
worker_cpu_policy = "logical"
shutdown_timeout = "3s"
[logging]
level = "info"
output = "stdout"
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
static_list = [%q]
`, bind, mode, maxRequests, backend.LocalAddr().String())
					_, log := startProxy(t, binary, configuration)
					eventually(t, "workers ready", func() bool { return len(readyPIDs(log)) == 2 })
					address, err := net.ResolveUDPAddr("udp", bind)
					if err != nil {
						t.Fatal(err)
					}
					client, err = net.DialUDP("udp", nil, address)
					if err != nil {
						t.Fatal(err)
					}
					clients.Add(2)
					go func() {
						defer clients.Done()
						payload := make([]byte, 256)
						for {
							select {
							case <-stopped:
								return
							default:
								_, _ = client.Write(payload)
							}
						}
					}()
					go func() {
						defer clients.Done()
						buffer := make([]byte, 65535)
						for {
							if _, err := client.Read(buffer); err != nil {
								return
							}
							replies.Add(1)
						}
					}()
					eventually(t, "backend traffic", func() bool { return packets.Load() > 100 })
					if maxRequests == 0 {
						eventually(t, "response traffic", func() bool { return replies.Load() > 100 })
					}
					if iteration == 0 {
						time.Sleep(2200 * time.Millisecond)
					} else {
						time.Sleep(100 * time.Millisecond)
					}
				})
			}
		}
	}
}
