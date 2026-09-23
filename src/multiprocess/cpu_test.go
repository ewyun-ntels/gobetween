package multiprocess

import (
	"reflect"
	"testing"

	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/stats"
)

func TestSplitCPUsBalanced(t *testing.T) {
	got, err := splitCPUs([]int{7, 3, 5, 1, 9}, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]int{{1, 3, 5}, {7, 9}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCPUs() = %v, want %v", got, want)
	}
}

func TestSplitCPUsSharesWhenOversubscribed(t *testing.T) {
	got, err := splitCPUs([]int{2, 4}, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]int{{2}, {4}, {2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCPUs() = %v, want %v", got, want)
	}
}

func TestCPUListRoundTrip(t *testing.T) {
	want := []int{1, 4, 8}
	got, err := parseCPUList(formatCPUList(want))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CPU list = %v, want %v", got, want)
	}
}

func TestAggregateSnapshots(t *testing.T) {
	target := core.Target{Host: "10.0.0.1", Port: "53"}
	slots := []*workerSlot{
		{snapshot: map[string]stats.Stats{
			"dns": {
				ActiveConnections: 2,
				RxTotal:           10,
				Backends: []core.Backend{{
					Target: target,
					Stats:  core.BackendStats{Live: true, TotalConnections: 3, RxBytes: 10},
				}},
			},
		}},
		{snapshot: map[string]stats.Stats{
			"dns": {
				ActiveConnections: 4,
				RxTotal:           20,
				Backends: []core.Backend{{
					Target: target,
					Stats:  core.BackendStats{TotalConnections: 5, RxBytes: 20},
				}},
			},
		}},
	}

	got := aggregateSnapshots(slots)["dns"]
	if got.ActiveConnections != 6 || got.RxTotal != 30 {
		t.Fatalf("aggregate server stats = %#v", got)
	}
	if len(got.Backends) != 1 || got.Backends[0].Stats.TotalConnections != 8 || got.Backends[0].Stats.RxBytes != 30 || !got.Backends[0].Stats.Live {
		t.Fatalf("aggregate backend stats = %#v", got.Backends)
	}
}
