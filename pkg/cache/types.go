package cache

import (
	"errors"
	"time"

	"github.com/benbjohnson/clock"
)

var ErrCacheWrongType error = errors.New("requested cache exists with that name, but a different type")
var ErrCacheTooCostly error = errors.New("item too costly for cache")
var ErrCacheFull error = errors.New("cache is full")

type CacheOptions struct {
	maxCost          uint64
	cleanupFrequency clock.Duration
	nowFactory       func() time.Time
	tickerFactory    func(clock.Duration) *clock.Ticker
}

// NewCacheOptions creates options describing a new in-memory cache.
//
// expectedItemTotal - gives an indication of how many items we expect
// to hold (maximum).
//
// maximumCost - is the actual cache capacity measured in some unit,
// where that unit is controlled by whoever is writing values to
// the cache. e.g. if we set this to 1048576 (bytes) and each
// write specifies it's size, then we have implemented a 1MiB
// maximum capacity.
func NewCacheOptions(
	maximumCost uint64,
	cleanupFrequency time.Duration,
) CacheOptions {
	clock := clock.New()
	return NewCacheOptionsWithFactories(
		maximumCost, cleanupFrequency, clock.Ticker, clock.Now,
	)
}

func NewCacheOptionsWithFactories(
	maximumCost uint64,
	cleanupFrequency time.Duration,
	tickerFunc func(clock.Duration) *clock.Ticker,
	nowFunc func() time.Time,
) CacheOptions {
	return CacheOptions{
		maxCost:          maximumCost,
		cleanupFrequency: cleanupFrequency,
		tickerFactory:    tickerFunc,
		nowFactory:       nowFunc,
	}
}
