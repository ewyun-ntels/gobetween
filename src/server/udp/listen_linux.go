//go:build linux

package udp

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func listenUDP(address string, reusePort bool) (*net.UDPConn, error) {
	lc := net.ListenConfig{}
	if reusePort {
		lc.Control = func(network, address string, raw syscall.RawConn) error {
			var controlErr error
			if err := raw.Control(func(fd uintptr) {
				controlErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			}); err != nil {
				return err
			}
			return controlErr
		}
	}

	packetConn, err := lc.ListenPacket(context.Background(), "udp", address)
	if err != nil {
		return nil, err
	}

	udpConn, ok := packetConn.(*net.UDPConn)
	if !ok {
		packetConn.Close()
		return nil, fmt.Errorf("unexpected UDP listener type %T", packetConn)
	}
	return udpConn, nil
}
