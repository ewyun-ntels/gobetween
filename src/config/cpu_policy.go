package config

import "fmt"

// CPUAllocationPolicy resolves the default without changing the loaded config.
// Physical allocation is only meaningful in the CPU-pinned worker runtime.
func CPUAllocationPolicy(runtime RuntimeConfig) (string, error) {
	switch runtime.WorkerCPUPolicy {
	case "", "logical":
		return "logical", nil
	case "physical":
		if runtime.WorkerProcesses <= 0 {
			return "", fmt.Errorf("runtime.worker_cpu_policy = physical requires worker_processes > 0")
		}
		return "physical", nil
	default:
		return "", fmt.Errorf("invalid runtime.worker_cpu_policy %q: expected logical or physical", runtime.WorkerCPUPolicy)
	}
}
