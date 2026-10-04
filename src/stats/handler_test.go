package stats

import (
	"testing"
	"time"

	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/core"
	"github.com/yyyar/gobetween/metrics"
)

func TestStopWithPendingTraffic(t *testing.T) {
	metrics.Start(config.MetricsConfig{})
	h := NewHandler(t.Name())
	h.Start()
	target := core.Target{Host: "127.0.0.1", Port: "5000"}
	h.BackendsCounter.In <- []core.Target{target}
	// No consumer of backend output: after a tick the forwarding path
	// can block. Stop must cancel that path before stopping the counters.
	h.Traffic <- core.ReadWriteCount{Target: target, CountWrite: 10}
	time.Sleep(INTERVAL + 50*time.Millisecond)
	for i := 0; i < 100; i++ {
		h.Traffic <- core.ReadWriteCount{Target: target, CountWrite: 10}
	}
	done := make(chan struct{})
	go func() { h.Stop(); h.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("statistics shutdown blocked on pending forwarding/output")
	}
}
