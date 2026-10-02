package config

import "fmt"

// UDPDistribution returns the receive-worker policy, not the backend balancer.
// An omitted value deliberately preserves the kernel's default selection.
func UDPDistribution(server Server) (string, error) {
	if server.Udp == nil || server.Udp.ReusePortDistribution == "" {
		return "hash", nil
	}
	if server.Protocol != "udp" {
		return "", fmt.Errorf("reuse_port_distribution requires protocol = udp")
	}
	switch server.Udp.ReusePortDistribution {
	case "hash", "rr":
		return server.Udp.ReusePortDistribution, nil
	default:
		return "", fmt.Errorf("invalid reuse_port_distribution %q: expected hash or rr", server.Udp.ReusePortDistribution)
	}
}

// ValidateUDPDistributions also catches rr in the legacy runtime before a
// listener or a management endpoint is opened.
func ValidateUDPDistributions(cfg Config) error {
	for name, server := range cfg.Servers {
		mode, err := UDPDistribution(server)
		if err != nil {
			return fmt.Errorf("server %q: %w", name, err)
		}
		if mode == "rr" && cfg.Runtime.WorkerProcesses <= 0 {
			return fmt.Errorf("server %q: rr requires runtime.worker_processes > 0", name)
		}
	}
	return nil
}
