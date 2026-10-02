// Package udpdispatch owns optional kernel-side UDP receive-worker selection.
// It never receives or forwards application packets in the supervisor.
package udpdispatch

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/yyyar/gobetween/config"
)

const (
	ListenersEnv   = "GOBETWEEN_UDP_LISTENERS"
	WorkerCountEnv = "GOBETWEEN_WORKER_PROCESSES"
	MaxWorkers     = 256
)

type InheritedListener struct {
	FD   int    `json:"fd"`
	Bind string `json:"bind"`
}

// Controller is used only on the supervisor event loop. Ready publishes a
// socket after server initialization; Remove is idempotent and excludes it
// before closing any supervisor-owned descriptor.
type Controller interface {
	Prepare(int) ([]*os.File, map[string]InheritedListener, error)
	Ready(int) error
	Remove(int) error
	Close() error
}

func inheritedDescriptors() (map[string]InheritedListener, error) {
	result := make(map[string]InheritedListener)
	if value := os.Getenv(ListenersEnv); value != "" {
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			return nil, fmt.Errorf("decode inherited UDP listeners: %w", err)
		}
	}
	seen := make(map[int]bool)
	for name, listener := range result {
		if listener.FD < 4 || listener.Bind == "" || seen[listener.FD] {
			return nil, fmt.Errorf("invalid inherited UDP listener %q", name)
		}
		seen[listener.FD] = true
	}
	return result, nil
}

// ValidateInherited refuses configuration changes that would make a child
// ignore its inherited rr sockets. Children still use the existing config file.
func ValidateInherited(cfg config.Config) error {
	listeners, err := inheritedDescriptors()
	if err != nil {
		return err
	}
	for name, listener := range listeners {
		server, ok := cfg.Servers[name]
		mode, modeErr := config.UDPDistribution(server)
		if !ok || modeErr != nil || mode != "rr" || server.Bind != listener.Bind {
			return fmt.Errorf("inherited UDP listener %q does not match worker configuration", name)
		}
	}
	count, err := strconv.Atoi(os.Getenv(WorkerCountEnv))
	if err != nil || count <= 0 {
		return fmt.Errorf("invalid inherited worker count")
	}
	if count > 1 {
		for name, server := range cfg.Servers {
			mode, err := config.UDPDistribution(server)
			if err != nil {
				return err
			}
			if mode == "rr" {
				if _, ok := listeners[name]; !ok {
					return fmt.Errorf("missing inherited rr listener %q", name)
				}
			}
		}
	}
	return nil
}

func ListenInherited(name, bind string) (*net.UDPConn, bool, error) {
	listeners, err := inheritedDescriptors()
	if err != nil {
		return nil, false, err
	}
	listener, ok := listeners[name]
	if !ok {
		return nil, false, nil
	}
	if listener.Bind != bind {
		return nil, true, fmt.Errorf("inherited UDP bind mismatch for %q", name)
	}
	file := os.NewFile(uintptr(listener.FD), "gobetween-udp-"+name)
	connection, err := net.FilePacketConn(file)
	_ = file.Close()
	if err != nil {
		return nil, true, fmt.Errorf("inherit UDP socket %q: %w", name, err)
	}
	udpConn, ok := connection.(*net.UDPConn)
	if !ok {
		_ = connection.Close()
		return nil, true, fmt.Errorf("inherited listener %q is not UDP", name)
	}
	return udpConn, true, nil
}
