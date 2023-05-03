package cache

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/benbjohnson/clock"
)

type Cache[T any] struct {
	items         generic.SyncMap[string, CacheItem[T]]
	cost          Counter
	closer        chan struct{}
	nowFactory    func() time.Time
	tickerFactory func(clock.Duration) *clock.Ticker
}

type CacheItem[T any] struct {
	contents  T
	cost      uint64
	expiresAt int64
}

func NewCache[T any](options CacheOptions) (*Cache[T], error) {
	c := &Cache[T]{
		closer:        make(chan struct{}),
		cost:          NewCounter(options.maxCost),
		tickerFactory: options.tickerFactory,
		nowFactory:    options.nowFactory,
	}

	go c.cleanup(options.cleanupFrequency)
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

func (c *Cache[T]) cleanup(frequency clock.Duration) {
	ticker := c.tickerFactory(frequency)
	defer ticker.Stop()

	for {
		select {
		case <-c.closer:
			return
		case <-ticker.C:
			// Stop the ticker whilst we process evictions
			// otherwise we'll be constrained to finishing
			// evictions in <frequency
			ticker.Stop()
			c.evict()
			ticker.Reset(frequency)
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
