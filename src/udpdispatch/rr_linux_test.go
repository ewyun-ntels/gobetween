//go:build linux

package udpdispatch

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/yyyar/gobetween/config"
	"golang.org/x/sys/unix"
)

func rrConfig(bind string, workers int) config.Config {
	return config.Config{Runtime: config.RuntimeConfig{WorkerProcesses: workers}, Servers: map[string]config.Server{
		"proxy": {Protocol: "udp", Bind: bind, Udp: &config.Udp{ReusePortDistribution: "rr"}},
	}}
}

func TestHashAndSingleWorkerNeedNoBPF(t *testing.T) {
	cfg := rrConfig("127.0.0.1:4000", 4)
	cfg.Servers["proxy"].Udp.ReusePortDistribution = "hash"
	if c, err := NewController(cfg); err != nil || c != nil {
		t.Fatalf("hash: %v %v", c, err)
	}
	cfg.Servers["proxy"].Udp.ReusePortDistribution = "rr"
	cfg.Runtime.WorkerProcesses = 1
	if c, err := NewController(cfg); err != nil || c != nil {
		t.Fatalf("single worker: %v %v", c, err)
	}
}

func TestRRRejectsAmbiguousBindAndWorkerLimit(t *testing.T) {
	cfg := rrConfig("127.0.0.1:4000", MaxWorkers+1)
	if _, err := NewController(cfg); err == nil {
		t.Fatal("accepted too many workers")
	}
	cfg.Runtime.WorkerProcesses = 4
	cfg.Servers["second"] = config.Server{Protocol: "udp", Bind: "127.0.0.1:4000"}
	if _, err := NewController(cfg); err == nil {
		t.Fatal("accepted hash and rr sharing a bind")
	}
	delete(cfg.Servers, "second")
	server := cfg.Servers["proxy"]
	server.Bind = "127.0.0.1:0"
	cfg.Servers["proxy"] = server
	if _, err := NewController(cfg); err == nil {
		t.Fatal("accepted ephemeral rr port")
	}
}

func kernelController(t *testing.T, address string, workers int) (*controller, []*net.UDPConn) {
	t.Helper()
	if os.Geteuid() != 0 && os.Getenv("GOBETWEEN_REQUIRE_BPF") != "1" {
		t.Skip("kernel RR integration needs BPF privileges; set GOBETWEEN_REQUIRE_BPF=1 to require execution")
	}
	probe, err := net.ListenPacket("udp", address)
	if err != nil {
		t.Skipf("test address unavailable: %v", err)
	}
	bind := probe.LocalAddr().String()
	_ = probe.Close()
	ctrl, err := NewController(rrConfig(bind, workers))
	if err != nil {
		t.Fatalf("kernel RR initialization: %+v", err)
	}
	c := ctrl.(*controller)
	t.Cleanup(func() { _ = c.Close() })
	connections := make([]*net.UDPConn, workers)
	for id := 1; id <= workers; id++ {
		connections[id-1] = inheritTestSocket(t, c, id)
	}
	t.Cleanup(func() {
		for _, conn := range connections {
			if conn != nil {
				_ = conn.Close()
			}
		}
	})
	return c, connections
}

func inheritTestSocket(t *testing.T, c *controller, id int) *net.UDPConn {
	t.Helper()
	files, descriptors, err := c.Prepare(id)
	if err != nil {
		t.Fatal(err)
	}
	if descriptors["proxy"].FD != 4 || len(files) != 1 {
		t.Fatalf("bad inherited descriptors: %v", descriptors)
	}
	pc, err := net.FilePacketConn(files[0])
	if err != nil {
		t.Fatal(err)
	}
	conn := pc.(*net.UDPConn)
	if err := c.Ready(id); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var flags int
	if controlErr := raw.Control(func(fd uintptr) { flags, err = unix.FcntlInt(fd, unix.F_GETFL, 0) }); controlErr != nil {
		t.Fatal(controlErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	if flags&unix.O_NONBLOCK == 0 {
		t.Fatal("RR registration changed inherited socket to blocking mode")
	}
	return conn
}

func fixedClient(t *testing.T, connection *net.UDPConn) *net.UDPConn {
	t.Helper()
	client, err := net.DialUDP("udp", nil, connection.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func checkPacket(t *testing.T, client, receiver *net.UDPConn, sequence int) {
	t.Helper()
	payload := append([]byte{0, 255, 0, 128}, []byte(fmt.Sprintf("packet-%d", sequence))...)
	if _, err := client.Write(payload); err != nil {
		t.Fatal(err)
	}
	_ = receiver.SetReadDeadline(time.Now().Add(time.Second))
	buffer := make([]byte, 65535)
	n, _, err := receiver.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("packet %d was not sent to expected worker: %v", sequence, err)
	}
	if !bytes.Equal(buffer[:n], payload) {
		t.Fatalf("packet %d corrupted: %x", sequence, buffer[:n])
	}
}

func TestKernelRRFixedFlow(t *testing.T) {
	_, connections := kernelController(t, "127.0.0.1:0", 4)
	client := fixedClient(t, connections[0])
	for packet := 0; packet < 64; packet++ {
		checkPacket(t, client, connections[packet%4], packet)
	}
}

func TestKernelRRIPv6(t *testing.T) {
	_, connections := kernelController(t, "[::1]:0", 2)
	client := fixedClient(t, connections[0])
	for packet := 0; packet < 16; packet++ {
		checkPacket(t, client, connections[packet%2], packet)
	}
}

func TestRRPermissionFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("negative permission test requires an unprivileged user")
	}
	c, err := NewController(rrConfig("127.0.0.1:4000", 4))
	if err == nil {
		_ = c.Close()
		t.Skip("user has BPF privileges")
	}
	if !errors.Is(err, unix.EPERM) && !errors.Is(err, unix.EACCES) {
		t.Fatalf("unexpected permission failure: %v", err)
	}
}

func TestKernelRRDatagramBoundaries(t *testing.T) {
	_, connections := kernelController(t, "127.0.0.1:0", 4)
	client := fixedClient(t, connections[0])
	for sequence, size := range []int{0, 1, 1472, 4096, 65507} {
		payload := bytes.Repeat([]byte{byte(sequence)}, size)
		if _, err := client.Write(payload); err != nil {
			t.Fatal(err)
		}
		receiver := connections[sequence%4]
		_ = receiver.SetReadDeadline(time.Now().Add(time.Second))
		buffer := make([]byte, 65535)
		n, _, err := receiver.ReadFromUDP(buffer)
		if err != nil || !bytes.Equal(buffer[:n], payload) {
			t.Fatalf("datagram size %d: length=%d err=%v", size, n, err)
		}
	}
}

func TestKernelRRCloseRemoveAndRestart(t *testing.T) {
	c, connections := kernelController(t, "127.0.0.1:0", 4)
	client := fixedClient(t, connections[0])
	// Socket removal changes the kernel's compact socket indices. BPF keys
	// remain stable worker IDs, and stale snapshots retry another valid socket.
	_ = connections[1].Close()
	for packet, worker := range []int{0, 2, 2, 3} {
		checkPacket(t, client, connections[worker], packet)
	}
	if err := c.Remove(2); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(2); err != nil {
		t.Fatalf("Remove not idempotent: %v", err)
	}
	remaining := []int{0, 2, 3}
	for packet := 4; packet < 16; packet++ {
		checkPacket(t, client, connections[remaining[packet%3]], packet)
	}
	connections[1] = inheritTestSocket(t, c, 2)
	for packet := 16; packet < 32; packet++ {
		checkPacket(t, client, connections[packet%4], packet)
	}
}

func TestKernelRRConcurrentSenders(t *testing.T) {
	_, connections := kernelController(t, "127.0.0.1:0", 4)
	const senders, perSender = 8, 200
	var readers, writers sync.WaitGroup
	errors := make(chan error, senders+4)
	for _, connection := range connections {
		_ = connection.SetReadBuffer(1 << 20)
		readers.Add(1)
		go func(conn *net.UDPConn) {
			defer readers.Done()
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			buffer := make([]byte, 128)
			for n := 0; n < senders*perSender/4; n++ {
				if _, _, err := conn.ReadFromUDP(buffer); err != nil {
					errors <- err
					return
				}
			}
		}(connection)
	}
	for sender := 0; sender < senders; sender++ {
		client := fixedClient(t, connections[0])
		writers.Add(1)
		go func(conn *net.UDPConn) {
			defer writers.Done()
			for packet := 0; packet < perSender; packet++ {
				if _, err := conn.Write([]byte("independent")); err != nil {
					errors <- err
					return
				}
			}
		}(client)
	}
	writers.Wait()
	readers.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent RR distribution: %v", err)
	}
}

func TestKernelRRNoReadyWorkerDrops(t *testing.T) {
	c, connections := kernelController(t, "127.0.0.1:0", 4)
	for id := 1; id <= 4; id++ {
		if err := c.Remove(id); err != nil {
			t.Fatal(err)
		}
	}
	client := fixedClient(t, connections[0])
	if _, err := client.Write([]byte("must-not-fall-back-to-hash")); err != nil {
		t.Fatal(err)
	}
	// Keep the excluded worker sockets open: a hash fallback would queue the
	// packet on one of them or on the parent's unselectable anchor.
	for _, conn := range append(connections, c.groups["proxy"].anchor) {
		_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		if _, _, err := conn.ReadFromUDP(make([]byte, 128)); err == nil {
			t.Fatal("no-ready packet reached an excluded socket")
		} else if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
			t.Fatalf("unexpected receive error: %v", err)
		}
	}
}

func TestKernelRRServerIsolation(t *testing.T) {
	if os.Geteuid() != 0 && os.Getenv("GOBETWEEN_REQUIRE_BPF") != "1" {
		t.Skip("kernel RR needs BPF privileges")
	}
	first, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	second, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		_ = first.Close()
		t.Fatal(err)
	}
	cfg := rrConfig(first.LocalAddr().String(), 4)
	cfg.Servers["second"] = config.Server{Protocol: "udp", Bind: second.LocalAddr().String(), Udp: &config.Udp{ReusePortDistribution: "rr"}}
	_ = first.Close()
	_ = second.Close()
	ctrl, err := NewController(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c := ctrl.(*controller)
	defer c.Close()
	connections := make(map[string][]*net.UDPConn)
	for worker := 1; worker <= 4; worker++ {
		files, descriptors, err := c.Prepare(worker)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 || descriptors["proxy"].FD != 4 || descriptors["second"].FD != 5 {
			t.Fatalf("incorrect multi-server FD mapping: %v", descriptors)
		}
		for index, name := range c.names {
			connection, err := net.FilePacketConn(files[index])
			if err != nil {
				t.Fatal(err)
			}
			conn := connection.(*net.UDPConn)
			defer conn.Close()
			connections[name] = append(connections[name], conn)
		}
		if err := c.Ready(worker); err != nil {
			t.Fatal(err)
		}
	}
	firstClient := fixedClient(t, connections["proxy"][0])
	secondClient := fixedClient(t, connections["second"][0])
	for packet := 0; packet < 3; packet++ {
		checkPacket(t, firstClient, connections["proxy"][packet], packet)
	}
	for packet := 0; packet < 4; packet++ {
		checkPacket(t, secondClient, connections["second"][packet], packet)
	}
	checkPacket(t, firstClient, connections["proxy"][3], 3)
}
