//go:build linux

package udp

import "testing"

func TestListenUDPReusePort(t *testing.T) {
	first, err := listenUDP("127.0.0.1:0", true)
	if err != nil {
		t.Fatalf("first listenUDP() error = %v", err)
	}
	defer first.Close()

	second, err := listenUDP(first.LocalAddr().String(), true)
	if err != nil {
		t.Fatalf("second listenUDP() with SO_REUSEPORT error = %v", err)
	}
	defer second.Close()
}
