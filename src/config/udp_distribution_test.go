package config

import (
	"github.com/burntsushi/toml"
	"testing"
)

func TestUDPDistribution(t *testing.T) {
	for _, value := range []string{"", "hash", "rr", "random", "RR"} {
		t.Run(value, func(t *testing.T) {
			server := Server{Protocol: "udp", Udp: &Udp{ReusePortDistribution: value}}
			got, err := UDPDistribution(server)
			valid := value == "" || value == "hash" || value == "rr"
			if (err == nil) != valid {
				t.Fatalf("UDPDistribution(%q) error=%v", value, err)
			}
			if value == "" && got != "hash" {
				t.Fatalf("default=%q", got)
			}
		})
	}
	if _, err := UDPDistribution(Server{Protocol: "tcp", Udp: &Udp{ReusePortDistribution: "rr"}}); err == nil {
		t.Fatal("accepted rr on TCP")
	}
	if mode, err := UDPDistribution(Server{Protocol: "udp"}); err != nil || mode != "hash" {
		t.Fatalf("nil UDP settings: %s %v", mode, err)
	}
}

func TestUDPDistributionTOML(t *testing.T) {
	var cfg Config
	_, err := toml.Decode("[runtime]\nworker_processes=4\n[servers.proxy]\nprotocol=\"udp\"\n[servers.proxy.udp]\nreuse_port_distribution=\"rr\"\nmax_requests=1\nmax_responses=0\n", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateUDPDistributions(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Servers["proxy"].Udp.ReusePortDistribution != "rr" {
		t.Fatal("TOML field not decoded")
	}
	cfg.Runtime.WorkerProcesses = 0
	if err := ValidateUDPDistributions(cfg); err == nil {
		t.Fatal("rr accepted without worker runtime")
	}
}
