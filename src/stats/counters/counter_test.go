package counters

import (
	"testing"
	"time"

	"github.com/yyyar/gobetween/core"
)

func TestCounterStopWithoutOutputReader(t *testing.T) {
	c := NewBandwidthCounter(time.Second, make(chan BandwidthStats))
	c.Start()
	c.Traffic <- core.ReadWriteCount{CountWrite: 1}
	time.Sleep(1100 * time.Millisecond)
	done := make(chan struct{})
	go func() { c.Stop(); c.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("counter Stop blocked on unread output")
	}
}
