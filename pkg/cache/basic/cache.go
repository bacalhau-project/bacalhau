package basic

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/cache"
	"github.com/bacalhau-project/bacalhau/pkg/cache/counter"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

const (
	DefaultTTLSeconds = int64(3600)
)

type BasicCache[T any] struct {
	items            generic.SyncMap[string, CacheItem[T]]
	cost             counter.Counter
	closer           chan struct{}
	evictionFunction EvictItemFunc
	defaultTTL       int64
}

type CacheItem[T any] struct {
	contents  T
	cost      uint64
	expiresAt int64
}

func NewCache[T any](options ...Option) (*BasicCache[T], error) {
	// initialize config with default values (these could be constants).
	config := &Config{
		maxCost:          1000,
		cleanupFrequency: time.Hour,
		evictionFunction: func(key string, cost uint64, expiresAt int64, now int64) bool {
			return expiresAt != 0 && expiresAt <= now
		},
		defaultTTL: DefaultTTLSeconds,
	}

	// override defaults with passed options.
	for _, opt := range options {
		opt(config)
	}

	c := &BasicCache[T]{
		closer:           make(chan struct{}),
		cost:             counter.NewCounter(config.maxCost),
		evictionFunction: config.evictionFunction,
		defaultTTL:       config.defaultTTL,
	}

	go c.cleanup(config.cleanupFrequency)
	return c, nil
}

func (c *BasicCache[T]) Get(key string) (T, bool) {
	result, exists := c.items.Get(key)
	if !exists {
		return *new(T), false
	}

	return result.contents, true
}

func (c *BasicCache[T]) SetWithDefaultTTL(key string, value T, cost uint64) error {
	return c.Set(key, value, cost, c.defaultTTL)
}

func (c *BasicCache[T]) Set(key string, value T, cost uint64, expiresInSeconds int64) error {
	expires := time.Now().Add(time.Duration(expiresInSeconds * int64(time.Second))).Unix()

	item := CacheItem[T]{
		contents:  value,
		cost:      cost,
		expiresAt: expires,
	}

	if !c.cost.HasSpaceFor(item.cost) {
		return cache.ErrCacheTooCostly
	}

	c.cost.Inc(cost)
	c.items.Put(key, item)

	return nil
}

func (c *BasicCache[T]) Delete(key string) {
	c.items.Delete(key)
}

func (c *BasicCache[T]) Close() {
	close(c.closer)
}

func (c *BasicCache[T]) cleanup(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-c.closer:
			return
		case <-ticker.C:
			now := time.Now().Unix()
			c.items.Iter(func(key string, item CacheItem[T]) bool {
				if c.evictionFunction(key, item.cost, item.expiresAt, now) {
					c.items.Delete(key)
					c.cost.Dec(item.cost)
				}
				return true
			})
		} // end select
	}
}
