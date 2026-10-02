//go:build linux

package multiprocess

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readCoreCPUs(cpu int) ([]int, error) {
	return readCoreCPUsAt("/sys/devices/system/cpu", cpu)
}

func readCoreCPUsAt(root string, cpu int) ([]int, error) {
	path := filepath.Join(root, fmt.Sprintf("cpu%d", cpu), "topology", "core_cpus_list")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		path = filepath.Join(root, fmt.Sprintf("cpu%d", cpu), "topology", "thread_siblings_list")
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	list, err := parseTopologyCPUList(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return list, nil
}

// sysfs lists can contain ranges, unlike the worker's internal IPC CPU list.
func parseTopologyCPUList(value string) ([]int, error) {
	result := make([]int, 0)
	seen := make(map[int]bool)
	parseID := func(part string) (int, error) {
		part = strings.TrimSpace(part)
		if part == "" {
			return 0, fmt.Errorf("empty CPU ID")
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return 0, fmt.Errorf("invalid CPU ID %q", part)
			}
		}
		cpu, err := strconv.Atoi(part)
		// The existing affinity implementation uses a 1024-bit unix.CPUSet.
		if err != nil || cpu >= 1024 {
			return 0, fmt.Errorf("CPU ID %q exceeds the supported affinity range 0-1023", part)
		}
		return cpu, nil
	}
	for _, part := range strings.Split(strings.TrimSpace(value), ",") {
		bounds := strings.Split(part, "-")
		if len(bounds) > 2 {
			return nil, fmt.Errorf("invalid CPU range %q", part)
		}
		first, err := parseID(bounds[0])
		if err != nil {
			return nil, err
		}
		last := first
		if len(bounds) == 2 {
			last, err = parseID(bounds[1])
			if err != nil || last < first {
				return nil, fmt.Errorf("invalid CPU range %q", part)
			}
		}
		for cpu := first; cpu <= last; cpu++ {
			if seen[cpu] {
				return nil, fmt.Errorf("duplicate CPU %d in topology list", cpu)
			}
			seen[cpu] = true
			result = append(result, cpu)
		}
	}
	return result, nil
}
