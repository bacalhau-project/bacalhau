package cache

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/benbjohnson/clock"
)

type Cache[T any] struct {
	name         string
	items        generic.SyncMap[string, CacheItem[T]]
	cost         Counter
	closer       chan struct{}
	nowFactory   func() time.Time
	timerFactory func(clock.Duration) *clock.Timer
}

type CacheItem[T any] struct {
	contents  T
	cost      uint64
	expiresAt int64
}

func NewCache[T any](name string, options CacheOptions) (c *Cache[T], err error) {
	c = &Cache[T]{
		name:         name,
		closer:       make(chan struct{}),
		cost:         NewCounter(options.maxCost),
		timerFactory: options.timerFactory,
		nowFactory:   options.nowFactory,
	}

	go c.janitor(options.cleanupFrequency)
	return c, nil
}

func (c *Cache[T]) Get(key string) (T, bool) {
	result, exists := c.items.Get(key)
	if !exists {
		return *new(T), false
	}

	return result.contents, true
}

func (c *Cache[T]) Set(key string, value T, cost uint64, expiresInSeconds int64) error {
	expires := c.nowFactory().Add(clock.Duration(expiresInSeconds)).Unix()

	item := CacheItem[T]{
		contents:  value,
		cost:      cost,
		expiresAt: expires,
	}

	if !c.cost.HasSpaceFor(item.cost) {
		return ErrCacheTooCostly
	}

	c.cost.Inc(cost)
	c.items.Put(key, item)

	return nil
}

func (c *Cache[T]) Delete(key string) {
	c.items.Delete(key)
}

func (c *Cache[T]) Close() {
	close(c.closer)
}

func (c *Cache[T]) janitor(frequency clock.Duration) {
	timer := c.timerFactory(frequency)
	defer timer.Stop()

	for {
		select {
		case <-c.closer:
			return
		case <-timer.C:
			// Perform the evictions necessary and recreate the
			// timer. We specifically don't use a ticker to avoid
			//race conditions in the mock when having to stop and
			// reset it.
			c.evict()
			timer = c.timerFactory(frequency)
		}
	}
}

func (c *Cache[T]) evict() {
	now := c.nowFactory().Unix()
	c.items.Iter(func(key string, item CacheItem[T]) bool {
		//		fmt.Printf("E: %d, N: %d", item.expiresAt, now)
		if item.expiresAt != 0 && item.expiresAt <= now {
			c.items.Delete(key)
			c.cost.Dec(item.cost)
		}
		return true
	})
}
