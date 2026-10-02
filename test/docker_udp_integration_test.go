//go:build linux

package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in, local Linux Docker daemon only: --network=host must share the test's
// network namespace. No image is pushed and no cluster resources are created.
func dockerCommand(t *testing.T, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func testContainer(t *testing.T, image, configuration string, bpf bool) string {
	t.Helper()
	// os.TempDir may be an inaccessible go-test directory. Use a dedicated,
	// world-traversable directory with a read-only config for arbitrary UID.
	directory, err := os.MkdirTemp("/tmp", "gobetween-image-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "gobetween.toml"), []byte(configuration), 0444); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("gobetween-test-%d-%d", os.Getpid(), time.Now().UnixNano())
	args := []string{"run", "-d", "--name", name, "--label", "gobetween.local-test=true", "--network=host", "--read-only", "--user=1001290000:0", "--cap-drop=ALL", "--security-opt=no-new-privileges:true", "--cpus=8", "--memory=1g", "--mount", "type=bind,source=" + directory + ",target=/etc/gobetween/conf,readonly"}
	if cpus := os.Getenv("GOBETWEEN_DOCKER_CPUSET"); cpus != "" {
		args = append(args, "--cpuset-cpus="+cpus)
	}
	if bpf {
		// Docker 28's --user + --cap-add sets the bounding set but not the
		// effective/ambient sets. Use a TEST-ONLY handoff, not a privileged image.
		launcher := filepath.Join(directory, "ambient-launcher")
		build := exec.Command("go", "build", "-o", launcher, "./testdata/ambient-launcher")
		build.Env = append(os.Environ(), "CGO_ENABLED=0")
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build test launcher: %v\n%s", err, output)
		}
		if err := os.Chmod(launcher, 0755); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--user=0:0", "--cap-add=BPF", "--cap-add=SETUID", "--cap-add=SETGID", "--entrypoint=/etc/gobetween/conf/ambient-launcher")
	}
	// --entrypoint clears Docker's image CMD, so always pass the config args.
	args = append(args, image, "-c", "/etc/gobetween/conf/gobetween.toml")
	t.Cleanup(func() {
		// Only this test's uniquely named container is stopped/removed.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = exec.CommandContext(ctx, "docker", "stop", "--timeout=10", name).CombinedOutput()
		logs, _ := exec.CommandContext(ctx, "docker", "logs", name).CombinedOutput()
		if t.Failed() {
			t.Logf("container logs:\n%s", logs)
		}
		_, _ = exec.CommandContext(ctx, "docker", "rm", name).CombinedOutput()
	})
	dockerCommand(t, args...)
	return name
}

func TestDockerUDPIntegration(t *testing.T) {
	image := os.Getenv("GOBETWEEN_TEST_IMAGE")
	if image == "" {
		t.Skip("set GOBETWEEN_TEST_IMAGE for local Docker integration")
	}
	if _, err := exec.LookPath("helm"); err != nil {
		t.Fatal("helm is required to test the actual chart-generated config")
	}
	for _, tc := range []struct {
		name, mode     string
		responses, bpf bool
	}{
		{"hash_one_way", "hash", false, false},
		{"hash_request_response", "hash", true, false},
		{"rr_one_way", "rr", false, true},
		{"rr_request_response", "rr", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.bpf && os.Getenv("GOBETWEEN_DOCKER_BPF") != "1" {
				t.Skip("set GOBETWEEN_DOCKER_BPF=1 to approve CAP_BPF tests")
			}
			packets := make(chan backendPacket, 512)
			settings := []string{"discovery.kind=static", "udp.reusePortDistribution=" + tc.mode}
			for index := 0; index < 3; index++ {
				socket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = socket.Close() })
				settings = append(settings, fmt.Sprintf("discovery.staticList[%d]=%s", index, socket.LocalAddr()))
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
				}(index, socket)
			}
			bind, metrics := reserveUDP(t), reserveTCP(t)
			_, udpPort, _ := net.SplitHostPort(bind)
			_, metricsPort, _ := net.SplitHostPort(metrics)
			settings = append(settings, "udp.port="+udpPort, "metrics.port="+metricsPort, "udp.clientIdleTimeout=10s", "udp.backendIdleTimeout=10s")
			if tc.responses {
				settings = append(settings, "udp.maxRequests=0", "udp.maxResponses=0")
			}
			resources, _ := renderChart(t, settings...)
			var configuration string
			for _, resource := range resources {
				if resource.Kind == "ConfigMap" {
					configuration = resource.Data["gobetween.toml"]
				}
			}
			name := testContainer(t, image, configuration, tc.bpf)
			httpClient := &http.Client{Timeout: time.Second}
			eventually(t, "container workers and backend discovery", func() bool {
				response, err := httpClient.Get("http://" + metrics + "/metrics")
				if err != nil {
					return false
				}
				defer response.Body.Close()
				data, _ := io.ReadAll(response.Body)
				return len(regexp.MustCompile(`(?m)^gobetween_worker_up\{[^}]*\} 1$`).FindAll(data, -1)) == 4 && strings.Contains(string(data), "gobetween_backend_live{")
			})
			pid := dockerCommand(t, "inspect", "--format", "{{.State.Pid}}", name)
			status, err := os.ReadFile("/proc/" + pid + "/status")
			if err != nil {
				t.Fatal(err)
			}
			statusText := string(status)
			for _, expected := range []string{"Uid:\t1001290000\t1001290000\t1001290000\t1001290000", "NoNewPrivs:\t1", "Seccomp:\t2"} {
				if !strings.Contains(statusText, expected) {
					t.Fatalf("missing actual process protection %s\n%s", expected, statusText)
				}
			}
			capability := "CapEff:\t0000000000000000"
			if tc.bpf {
				capability = "CapEff:\t0000008000000000"
			}
			if !strings.Contains(statusText, capability) {
				t.Fatalf("unexpected effective capabilities\n%s", statusText)
			}
			address, _ := net.ResolveUDPAddr("udp", bind)
			client, err := net.DialUDP("udp", nil, address)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			peers := make(map[string]int)
			for sequence := 0; sequence < 120; sequence++ {
				payload := append([]byte{0, 255, 128, 0}, []byte("container-message-"+strconv.Itoa(sequence))...)
				if _, err := client.Write(payload); err != nil {
					t.Fatal(err)
				}
				select {
				case packet := <-packets:
					if !bytes.Equal(packet.payload, payload) {
						t.Fatal("backend payload mismatch")
					}
					peers[packet.peer]++
				case <-time.After(2 * time.Second):
					t.Fatal("missing UDP packet")
				}
				if tc.responses {
					_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
					buffer := make([]byte, 65535)
					n, err := client.Read(buffer)
					if err != nil || !bytes.Equal(buffer[:n], payload) {
						t.Fatalf("UDP response mismatch: %v", err)
					}
				}
			}
			wantPeers := 3
			if tc.bpf {
				wantPeers = 12
			}
			if tc.responses {
				wantPeers = 1
				if tc.bpf {
					wantPeers = 4
				}
			}
			if len(peers) != wantPeers {
				t.Fatalf("backend socket count=%d want %d", len(peers), wantPeers)
			}
			for peer, count := range peers {
				if count != 120/wantPeers {
					t.Errorf("peer %s count=%d want %d", peer, count, 120/wantPeers)
				}
			}
			dockerCommand(t, "stop", "--timeout=15", name)
			if exit := dockerCommand(t, "inspect", "--format", "{{.State.ExitCode}}", name); exit != "0" {
				t.Fatalf("nonzero container exit: %s", exit)
			}
			logs := dockerCommand(t, "logs", name)
			if !strings.Contains(logs, "stopping UDP workers") || strings.Contains(logs, "Force killing worker") {
				t.Fatal("SIGTERM did not shut down workers gracefully")
			}
		})
	}
	if os.Getenv("GOBETWEEN_DOCKER_BPF") == "1" {
		t.Run("rr_without_capability_fails", func(t *testing.T) {
			resources, _ := renderChart(t, "udp.port="+strings.Split(reserveUDP(t), ":")[1], "metrics.port="+strings.Split(reserveTCP(t), ":")[1])
			var configuration string
			for _, resource := range resources {
				if resource.Kind == "ConfigMap" {
					configuration = resource.Data["gobetween.toml"]
				}
			}
			name := testContainer(t, image, configuration, false)
			eventually(t, "RR startup failure without BPF", func() bool { return dockerCommand(t, "inspect", "--format", "{{.State.Running}}", name) == "false" })
			if exit := dockerCommand(t, "inspect", "--format", "{{.State.ExitCode}}", name); exit == "0" {
				t.Fatal("RR silently succeeded without BPF capability")
			}
			logs := dockerCommand(t, "logs", name)
			if !strings.Contains(logs, "operation not permitted") && !strings.Contains(logs, "permission denied") {
				t.Fatalf("unexpected startup failure:\n%s", logs)
			}
		})
	}
}
