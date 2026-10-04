package multiprocess

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/metrics"
	"github.com/yyyar/gobetween/stats"
	"github.com/yyyar/gobetween/udpdispatch"
)

type fakeDispatch struct {
	ready, removed []int
	failure        error
}

func TestSupervisorCPUConfig(t *testing.T) {
	for _, policy := range []string{"", "logical", "physical", "phygical", "core"} {
		cfg := config.Config{
			Runtime: config.RuntimeConfig{WorkerProcesses: 4, WorkerCPUPolicy: policy},
			Servers: map[string]config.Server{"udp": {Protocol: "udp"}},
		}
		err := validateSupervisorConfig(cfg)
		valid := policy == "" || policy == "logical" || policy == "physical"
		if (err == nil) != valid {
			t.Errorf("policy=%q err=%v", policy, err)
		}
	}
}

func (d *fakeDispatch) Prepare(int) ([]*os.File, map[string]udpdispatch.InheritedListener, error) {
	return nil, nil, nil
}
func (d *fakeDispatch) Ready(id int) error  { d.ready = append(d.ready, id); return d.failure }
func (d *fakeDispatch) Remove(id int) error { d.removed = append(d.removed, id); return d.failure }
func (d *fakeDispatch) Close() error        { return nil }

func TestDispatchWorkerLifecycle(t *testing.T) {
	metrics.Start(config.MetricsConfig{})
	dispatch := &fakeDispatch{}
	slot := &workerSlot{id: 1, generation: 2, dispatch: dispatch}
	slots := []*workerSlot{slot}
	events := make(chan workerEvent, 4)
	ready := workerEvent{workerID: 1, generation: 2, kind: eventMessage, message: Message{Type: messageReady}}
	stale := ready
	stale.generation = 1
	if err := handleWorkerEvent(stale, slots, events, false, time.Second); err != nil {
		t.Fatal(err)
	}
	if len(dispatch.ready) != 0 {
		t.Fatal("stale generation activated socket")
	}
	for repeat := 0; repeat < 2; repeat++ {
		if err := handleWorkerEvent(ready, slots, events, false, time.Second); err != nil {
			t.Fatal(err)
		}
	}
	if !slot.ready || len(dispatch.ready) != 1 {
		t.Fatal("ready was not idempotent")
	}
	if err := handleWorkerEvent(workerEvent{workerID: 1, generation: 2, kind: eventReadEnd}, slots, events, false, time.Second); err != nil {
		t.Fatal(err)
	}
	if slot.ready || len(dispatch.removed) != 1 {
		t.Fatal("IPC close did not exclude RR socket")
	}
}

func TestDispatchFailurePropagates(t *testing.T) {
	metrics.Start(config.MetricsConfig{})
	want := errors.New("BPF update failed")
	dispatch := &fakeDispatch{failure: want}
	slot := &workerSlot{id: 1, generation: 1, dispatch: dispatch}
	err := handleWorkerEvent(workerEvent{workerID: 1, generation: 1, kind: eventMessage, message: Message{Type: messageReady}}, []*workerSlot{slot}, make(chan workerEvent), false, time.Second)
	if !errors.Is(err, want) || slot.ready {
		t.Fatalf("activation failure: %v ready=%t", err, slot.ready)
	}
}

func TestPartialStartupShutdown(t *testing.T) {
	dispatch := &fakeDispatch{}
	shutdownWorkerSet([]*workerSlot{{id: 1, dispatch: dispatch}, nil}, make(chan workerEvent), time.Second)
	if len(dispatch.removed) != 1 {
		t.Fatal("partial startup did not clean RR sockets")
	}
}

func TestDispatchIgnoresMessagesAfterDisconnect(t *testing.T) {
	metrics.Start(config.MetricsConfig{})
	for _, terminal := range []string{eventExit, eventReadEnd} {
		t.Run(terminal, func(t *testing.T) {
			dispatch := &fakeDispatch{}
			slot := &workerSlot{id: 1, generation: 1, dispatch: dispatch}
			slots := []*workerSlot{slot}
			events := make(chan workerEvent, 4)
			if err := handleWorkerEvent(workerEvent{workerID: 1, generation: 1, kind: terminal}, slots, events, false, time.Second); err != nil {
				t.Fatal(err)
			}
			dispatch.failure = errors.New("no pending listener after removal")
			for _, kind := range []string{messageReady, messageHeartbeat, messageStats} {
				err := handleWorkerEvent(workerEvent{workerID: 1, generation: 1, kind: eventMessage, message: Message{
					Type: kind, Stats: map[string]stats.Stats{"proxy": {TxTotal: 100}},
				}}, slots, events, false, time.Second)
				if err != nil {
					t.Fatalf("late %s must not abort supervisor: %v", kind, err)
				}
			}
			if slot.ready || len(dispatch.ready) != 0 || slot.snapshot != nil || !slot.lastHeartbeat.IsZero() {
				t.Fatal("late IPC message revived disconnected worker")
			}
		})
	}
}
