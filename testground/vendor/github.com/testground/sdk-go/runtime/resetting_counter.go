package runtime

import (
	"sync/atomic"

	"github.com/rcrowley/go-metrics"
)

func newResettingCounter() Counter {
	if metrics.UseNilMetrics {
		return metrics.NilCounter{}
	}
	return &standardResettingCounter{0}
}

// StandardResettingCounter is the standard implementation of a Counter and uses the
// sync/atomic package to manage a single int64 value. It resets when Snapshot() is called.
type standardResettingCounter struct {
	count int64
}

// Clear sets the counter to zero.
func (c *standardResettingCounter) Clear() {
	atomic.StoreInt64(&c.count, 0)
}

// Count returns the current count.
func (c *standardResettingCounter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

// Dec decrements the counter by the given amount.
func (c *standardResettingCounter) Dec(i int64) {
	atomic.AddInt64(&c.count, -i)
}

// Inc increments the counter by the given amount.
func (c *standardResettingCounter) Inc(i int64) {
	atomic.AddInt64(&c.count, i)
}

// Snapshot returns a read-only copy of the counter, and resets it.
func (c *standardResettingCounter) Snapshot() Counter {
	currentValue := atomic.SwapInt64(&c.count, 0)
	return metrics.CounterSnapshot(currentValue)
}
