//go:build !linux

package udp

import (
	"fmt"
	"net"
)

func listenUDP(address string, reusePort bool) (*net.UDPConn, error) {
	if reusePort {
		return nil, fmt.Errorf("SO_REUSEPORT UDP workers are supported only on Linux")
	}
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", addr)
}
