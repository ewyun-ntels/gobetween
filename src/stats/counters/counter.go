package counters

/**
 * counter.go - bandwidth counter
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"sync"
	"time"

	"github.com/yyyar/gobetween/core"
)

/**
 * Count total bandwidth and bandwidth per second
 */
type BandwidthCounter struct {

	/* Bandwidth Stats */
	BandwidthStats

	/* Last received total bytes */
	RxTotalLast uint64
	/* Last transmitted total bytes */
	TxTotalLast uint64

	/* Timeframe to calculate per-second bandwidth */
	interval time.Duration
	/* Ticker for per-second bandwidth calculation and pushing stats */
	ticker *time.Ticker

	/* Indicates that new bandwidth delta was received */
	newTxRx bool

	/* ----- channels ----- */

	/* Input channel for bandwidth deltas */
	Traffic chan core.ReadWriteCount

	/* Stop channel */
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once

	/* Output channel for bandwidth stats */
	Out chan BandwidthStats
}

/**
 * Create new BandwidthCounter
 */
func NewBandwidthCounter(interval time.Duration, out chan BandwidthStats) *BandwidthCounter {

	return &BandwidthCounter{
		interval: interval,
		ticker:   time.NewTicker(interval),
		BandwidthStats: BandwidthStats{
			RxTotal: 0,
			TxTotal: 0,
		},
		TxTotalLast: 0,
		RxTotalLast: 0,
		Out:         out,
		Traffic:     make(chan core.ReadWriteCount),
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
}

/**
 * Starts bandwidth counter
 */
func (this *BandwidthCounter) Start() {

	go func() {
		defer close(this.done)
		defer this.ticker.Stop()

		for {
			select {

			// Stop requested
			case <-this.stop:
				return

				// New counting cycle
			case <-this.ticker.C:

				if !this.newTxRx {
					this.RxSecond = 0
					this.TxSecond = 0
				} else {

					dRx := this.RxTotal - this.RxTotalLast
					dTx := this.TxTotal - this.TxTotalLast

					this.RxSecond = uint(dRx / uint64(this.interval.Seconds()))
					this.TxSecond = uint(dTx / uint64(this.interval.Seconds()))

					this.RxTotalLast = this.RxTotal
					this.TxTotalLast = this.TxTotal

					this.newTxRx = false
				}

				// Send results to out
				select {
				case this.Out <- this.BandwidthStats:
				case <-this.stop:
					return
				}

				// New traffic deltas available
			case rwc := <-this.Traffic:
				this.newTxRx = true
				this.RxTotal += uint64(rwc.CountRead)
				this.TxTotal += uint64(rwc.CountWrite)
			}
		}
	}()
}

/**
 * Stops bandwidth counter
 */
func (this *BandwidthCounter) Stop() {
	this.stopOnce.Do(func() { close(this.stop) })
	<-this.done
}
