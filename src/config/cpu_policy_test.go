package config

import (
	"encoding/json"
	"testing"

	"github.com/burntsushi/toml"
)

func TestCPUAllocationPolicy(t *testing.T) {
	for _, test := range []struct {
		name    string
		policy  string
		workers int
		want    string
		fail    bool
	}{
		{"default legacy", "", 0, "logical", false},
		{"default workers", "", 4, "logical", false},
		{"logical", "logical", 4, "logical", false},
		{"physical", "physical", 4, "physical", false},
		{"physical single worker", "physical", 1, "physical", false},
		{"physical legacy", "physical", 0, "", true},
		{"physical negative workers", "physical", -1, "", true},
		{"typo", "phygical", 4, "", true},
		{"old proposal", "core", 4, "", true},
		{"case", "Physical", 4, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := CPUAllocationPolicy(RuntimeConfig{WorkerCPUPolicy: test.policy, WorkerProcesses: test.workers})
			if (err != nil) != test.fail || got != test.want {
				t.Fatalf("policy = %q, err = %v; want %q, fail=%v", got, err, test.want, test.fail)
			}
		})
	}
}

func TestWorkerCPUPolicyJSON(t *testing.T) {
	var cfg Config
	if err := json.Unmarshal([]byte(`{"runtime":{"worker_processes":4,"worker_cpu_policy":"physical"}}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if got, err := CPUAllocationPolicy(cfg.Runtime); err != nil || got != "physical" {
		t.Fatalf("decoded policy=%q, err=%v", got, err)
	}
}

func TestWorkerCPUPolicyTOML(t *testing.T) {
	var cfg Config
	if _, err := toml.Decode("[runtime]\nworker_processes = 4\nworker_cpu_policy = \"physical\"\n", &cfg); err != nil {
		t.Fatal(err)
	}
	if got, err := CPUAllocationPolicy(cfg.Runtime); err != nil || got != "physical" {
		t.Fatalf("decoded TOML policy=%q, err=%v", got, err)
	}
}
