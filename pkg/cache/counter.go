package cache

import "sync/atomic"

type Counter struct {
	// current must be first element in the struct to ensure
	// alignment on 32bit systems.
	current uint64
	maximum uint64
}

func NewCounter(max uint64) Counter {
	return Counter{
		current: 0,
		maximum: max,
	}
}

func (c *Counter) Inc(by uint64) {
	atomic.AddUint64(&c.current, by)
}

func (c *Counter) Dec(by uint64) {
	atomic.AddUint64(&c.current, ^uint64(by-1)) //nolint:unconvert
}

func (c *Counter) Current() uint64 {
	return atomic.LoadUint64(&c.current)
}

func (c *Counter) Reset(max uint64) {
	atomic.StoreUint64(&c.current, 0)
	c.maximum = max
}

func (c *Counter) HasSpaceFor(i uint64) bool {
	return c.Current()+i <= c.maximum
}

func (c *Counter) IsFull() bool {
	return c.Current() == c.maximum
}
