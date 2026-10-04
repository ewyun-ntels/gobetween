package counters

/**
 * backendscounter.go - bandwidth counter for backends pool
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"sync"
	"time"

	"github.com/yyyar/gobetween/core"
)

const (
	/* Stats update interval */
	INTERVAL = 2 * time.Second
)

/**
 * Bandwidth counter for backends pool
 */
type BackendsBandwidthCounter struct {

	/* Map of counters of specific targets */
	counters map[core.Target]*BandwidthCounter

	/* ----- channels ------ */

	/* Input channel of updated targets */
	In chan []core.Target

	/* Input channel of traffic deltas */
	Traffic chan core.ReadWriteCount

	/* Output channel for counted stats */
	Out chan BandwidthStats

	/* Stop channel */
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

/**
 * Creates new backends bandwidth counter
 */
func NewBackendsBandwidthCounter() *BackendsBandwidthCounter {
	return &BackendsBandwidthCounter{
		counters: make(map[core.Target]*BandwidthCounter),
		In:       make(chan []core.Target),
		Traffic:  make(chan core.ReadWriteCount),
		Out:      make(chan BandwidthStats),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

/**
 * Start backends counter
 */
func (this *BackendsBandwidthCounter) Start() {

	go func() {
		defer close(this.done)
		defer func() {
			for _, counter := range this.counters {
				counter.Stop()
			}
			this.counters = nil
		}()
		for {
			select {

			// stop
			case <-this.stop:

				return

			// new backends available
			case targets := <-this.In:
				this.UpdateCounters(targets)

			// new traffic available
			// route to appropriated counter
			case rwc := <-this.Traffic:
				counter, ok := this.counters[rwc.Target]
				// ignore stats for backend that is not is list
				if ok {
					select {
					case counter.Traffic <- rwc:
					case <-this.stop:
						return
					}
				}
			}

		}
	}()
}

/**
 * Update counters to match targets, optionally creating new
 * and deleting old counters
 */
func (this *BackendsBandwidthCounter) UpdateCounters(targets []core.Target) {

	result := map[core.Target]*BandwidthCounter{}

	// Keep or add needed workers
	for _, t := range targets {
		c, ok := this.counters[t]
		if !ok {
			c = NewBandwidthCounter(INTERVAL, this.Out)
			c.Target = t
			c.Start()
		}
		result[t] = c
	}

	// Stop needed counters
	for currentT, c := range this.counters {
		remove := true
		for _, t := range targets {
			if currentT.EqualTo(t) {
				remove = false
				break
			}
		}

		if remove {
			c.Stop()
		}
	}

	this.counters = result
}

/**
 * Stop backends counter
 */
func (this *BackendsBandwidthCounter) Stop() {
	this.stopOnce.Do(func() { close(this.stop) })
	<-this.done
}
