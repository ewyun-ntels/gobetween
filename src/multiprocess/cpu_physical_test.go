package multiprocess

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPhysicalCPUPartitions(t *testing.T) {
	for _, test := range []struct {
		name    string
		groups  [][]int
		workers int
		want    [][]int
	}{
		{"SMT48", [][]int{{0, 48}, {1, 49}, {2, 50}, {3, 51}}, 4, [][]int{{0, 48}, {1, 49}, {2, 50}, {3, 51}}},
		{"SMT96", [][]int{{8, 104}, {9, 105}, {10, 106}, {11, 107}}, 4, [][]int{{8, 104}, {9, 105}, {10, 106}, {11, 107}}},
		{"SMT4", [][]int{{0, 4, 8, 12}, {1, 5, 9, 13}}, 2, [][]int{{0, 4, 8, 12}, {1, 5, 9, 13}}},
		{"uneven cores", [][]int{{0, 48}, {1, 49}, {2, 50}, {3, 51}, {4, 52}}, 2, [][]int{{0, 1, 2, 48, 49, 50}, {3, 4, 51, 52}}},
		{"no SMT", [][]int{{7}, {2}, {99}}, 2, [][]int{{2, 7}, {99}}},
		{"noncontiguous IDs", [][]int{{17, 3}, {101, 9}}, 2, [][]int{{3, 17}, {9, 101}}},
		{"one worker", [][]int{{0, 48}, {1, 49}}, 1, [][]int{{0, 1, 48, 49}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			available := []int{}
			topology := make(map[int][]int)
			for _, group := range test.groups {
				for _, cpu := range group {
					available = append(available, cpu)
					topology[cpu] = group
				}
			}
			original := append([]int(nil), available...)
			reader := func(cpu int) ([]int, error) { return topology[cpu], nil }
			got, err := partitionWorkerCPUs(available, test.workers, "physical", reader)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("partitions=%v err=%v, want=%v", got, err, test.want)
			}
			if !reflect.DeepEqual(available, original) {
				t.Fatal("input CPU list was modified")
			}
			owners := make(map[int]int)
			for worker, cpus := range got {
				for _, cpu := range cpus {
					if _, exists := owners[cpu]; exists {
						t.Fatalf("CPU %d assigned twice", cpu)
					}
					owners[cpu] = worker
				}
			}
			for _, group := range test.groups {
				for _, cpu := range group {
					if owner, ok := owners[cpu]; !ok || owner != owners[group[0]] {
						t.Fatalf("SMT group split or CPU missing: %v, owners=%v", group, owners)
					}
				}
			}
		})
	}
}

func TestPhysicalCPUFailures(t *testing.T) {
	for _, test := range []struct {
		name      string
		available []int
		workers   int
		topology  map[int][]int
	}{
		{"no CPUs", nil, 1, nil},
		{"no workers", []int{0}, 0, map[int][]int{0: {0}}},
		{"too many workers", []int{0, 48}, 2, map[int][]int{0: {0, 48}, 48: {0, 48}}},
		{"partial SMT", []int{0}, 1, map[int][]int{0: {0, 48}}},
		{"missing topology", []int{0}, 1, nil},
		{"empty topology", []int{0}, 1, map[int][]int{0: {}}},
		{"missing self", []int{0, 1}, 1, map[int][]int{0: {1}, 1: {1}}},
		{"inconsistent siblings", []int{0, 48}, 1, map[int][]int{0: {0, 48}, 48: {48}}},
		{"overlapping groups", []int{0, 1, 2}, 1, map[int][]int{0: {0, 1}, 1: {1, 2}, 2: {1, 2}}},
		{"duplicate member", []int{0}, 1, map[int][]int{0: {0, 0}}},
		{"duplicate allowed", []int{0, 0}, 1, map[int][]int{0: {0}}},
		{"negative allowed", []int{-1}, 1, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := func(cpu int) ([]int, error) {
				group, ok := test.topology[cpu]
				if !ok {
					return nil, fmt.Errorf("topology unavailable")
				}
				return group, nil
			}
			if got, err := partitionWorkerCPUs(test.available, test.workers, "physical", reader); err == nil {
				t.Fatalf("expected failure, got %v", got)
			}
		})
	}
	if _, err := splitPhysicalCPUs([]int{0}, 1, nil); err == nil {
		t.Fatal("nil reader accepted")
	}
}

func TestLogicalPolicyDoesNotReadTopology(t *testing.T) {
	reader := func(cpu int) ([]int, error) { t.Fatal("logical mode read topology"); return nil, nil }
	for _, test := range []struct {
		cpus    []int
		workers int
		want    [][]int
	}{
		{[]int{7, 3, 5, 1, 9}, 2, [][]int{{1, 3, 5}, {7, 9}}},
		{[]int{2, 4}, 3, [][]int{{2}, {4}, {2}}},
	} {
		got, err := partitionWorkerCPUs(test.cpus, test.workers, "logical", reader)
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Fatalf("partitions=%v err=%v", got, err)
		}
	}
	if _, err := partitionWorkerCPUs([]int{0}, 1, "core", reader); err == nil {
		t.Fatal("unknown policy accepted")
	}
}
