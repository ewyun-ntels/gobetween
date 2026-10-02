package multiprocess

import (
	"fmt"
	"sort"
)

func partitionWorkerCPUs(available []int, workers int, policy string, siblings func(int) ([]int, error)) ([][]int, error) {
	switch policy {
	case "logical":
		// Do not read sysfs in legacy mode, even if it is inaccessible.
		return splitCPUs(available, workers)
	case "physical":
		return splitPhysicalCPUs(available, workers, siblings)
	default:
		return nil, fmt.Errorf("unknown worker CPU policy %q", policy)
	}
}

// splitPhysicalCPUs preserves complete SMT groups and refuses to share a core
// between workers. Partial or inconsistent topology is an error, not a fallback.
func splitPhysicalCPUs(available []int, workers int, siblings func(int) ([]int, error)) ([][]int, error) {
	if workers <= 0 || len(available) == 0 {
		return nil, fmt.Errorf("physical allocation requires available CPUs and worker_processes > 0")
	}
	if siblings == nil {
		return nil, fmt.Errorf("CPU topology reader is unavailable")
	}
	cpus := append([]int(nil), available...)
	sort.Ints(cpus)
	allowed := make(map[int]bool, len(cpus))
	for _, cpu := range cpus {
		if cpu < 0 || allowed[cpu] {
			return nil, fmt.Errorf("invalid or duplicate allowed CPU %d", cpu)
		}
		allowed[cpu] = true
	}
	groups := make([][]int, 0)
	seen := make(map[string]bool)
	owner := make(map[int]string, len(cpus))
	for _, cpu := range cpus {
		group, err := siblings(cpu)
		if err != nil {
			return nil, fmt.Errorf("read CPU %d topology: %w", cpu, err)
		}
		group = append([]int(nil), group...)
		sort.Ints(group)
		containsSelf := false
		for index, member := range group {
			if index > 0 && member == group[index-1] {
				return nil, fmt.Errorf("duplicate CPU %d in CPU %d topology", member, cpu)
			}
			if !allowed[member] {
				return nil, fmt.Errorf("incomplete physical core for CPU %d: sibling CPU %d is outside the allowed CPU set", cpu, member)
			}
			containsSelf = containsSelf || member == cpu
		}
		if !containsSelf {
			return nil, fmt.Errorf("CPU %d topology does not include itself", cpu)
		}
		key := formatCPUList(group)
		for _, member := range group {
			if previous, ok := owner[member]; ok && previous != key {
				return nil, fmt.Errorf("inconsistent physical core topology for CPU %d: %s versus %s", member, previous, key)
			}
			owner[member] = key
		}
		if !seen[key] {
			seen[key] = true
			groups = append(groups, group)
		}
	}
	if len(groups) < workers {
		return nil, fmt.Errorf("worker_processes=%d exceeds available physical cores=%d (logical CPUs=%d)", workers, len(groups), len(cpus))
	}
	// Reuse the balanced partitioner for group indices, never for SMT members.
	indices := make([]int, len(groups))
	for index := range indices {
		indices[index] = index
	}
	assignments, err := splitCPUs(indices, workers)
	if err != nil {
		return nil, err
	}
	result := make([][]int, workers)
	for worker, assignment := range assignments {
		for _, index := range assignment {
			result[worker] = append(result[worker], groups[index]...)
		}
		sort.Ints(result[worker])
	}
	return result, nil
}
