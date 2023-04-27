package cache

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

type Cache[T any] struct {
	name     string
	items    generic.SyncMap[string, CacheItem[T]]
	cost     int64
	count    int64
	maxCost  int64
	maxItems int64
	closer   chan struct{}
}

type CacheItem[T any] struct {
	contents  T
	cost      int64
	expiresAt int64
}

var caches map[string]interface{} = make(map[string]interface{})

// GetOrCreateCache
func GetOrCreateCache[T any](name string, options CacheOptions) (*Cache[T], error) {
	if cache, ok := caches[name]; ok {
		if cast, ok := cache.(*Cache[T]); ok {
			return cast, nil
		}
		return nil, errWrongCacheType
	}

	cache, err := NewCache[T](name, options)
	if err != nil {
		return nil, err
	}

	caches[name] = cache
	return cache, nil
}

func NewCache[T any](name string, options CacheOptions) (c *Cache[T], err error) {
	c = &Cache[T]{
		name:     name,
		closer:   make(chan struct{}),
		cost:     0,
		count:    0,
		maxCost:  options.maxCost,
		maxItems: options.maxItems,
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

func (c *Cache[T]) Set(key string, value T, cost int64, ttl time.Duration) error {
	expires := time.Now().Add(ttl).Unix()

	item := CacheItem[T]{
		contents:  value,
		cost:      cost,
		expiresAt: expires,
	}

	if item.cost+c.cost > c.maxCost {
		return errTooCostly
	}

	if c.count == c.maxItems {
		return errTooFull
	}

	c.count += 1
	c.cost += cost
	c.items.Put(key, item)

	return nil
}

func (c *Cache[T]) Delete(key string) {
	c.items.Delete(key)
}

func (c *Cache[T]) cleanup(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
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
	now := time.Now().Unix()

	c.items.Iter(func(key string, item CacheItem[T]) bool {
		if item.expiresAt != 0 && item.expiresAt <= now {
			c.items.Delete(key)
			c.count -= 1
		}
		return true
	})
}
