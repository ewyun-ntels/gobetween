package multiprocess

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func splitCPUs(available []int, workers int) ([][]int, error) {
	if workers <= 0 {
		return nil, fmt.Errorf("worker_processes must be greater than zero")
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no CPUs are available to the process")
	}

	cpus := append([]int(nil), available...)
	sort.Ints(cpus)
	result := make([][]int, workers)
	if workers > len(cpus) {
		for worker := range result {
			result[worker] = []int{cpus[worker%len(cpus)]}
		}
		return result, nil
	}

	base := len(cpus) / workers
	extra := len(cpus) % workers
	offset := 0
	for worker := range result {
		count := base
		if worker < extra {
			count++
		}
		result[worker] = append([]int(nil), cpus[offset:offset+count]...)
		offset += count
	}
	return result, nil
}

func formatCPUList(cpus []int) string {
	parts := make([]string, len(cpus))
	for i, cpu := range cpus {
		parts[i] = strconv.Itoa(cpu)
	}
	return strings.Join(parts, ",")
}

func parseCPUList(value string) ([]int, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("empty CPU list")
	}
	parts := strings.Split(value, ",")
	result := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		cpu, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || cpu < 0 {
			return nil, fmt.Errorf("invalid CPU %q", part)
		}
		if _, ok := seen[cpu]; ok {
			continue
		}
		seen[cpu] = struct{}{}
		result = append(result, cpu)
	}
	return result, nil
}
