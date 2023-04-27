package cache

import (
	"errors"
	"time"
)

var errWrongCacheType error = errors.New("requested cache type did not match previous cache with that name")
var errTooCostly error = errors.New("item too costly for cache")
var errTooFull error = errors.New("cache is full")

type CacheOptions struct {
	maxItems         int64
	maxCost          int64
	cleanupFrequency time.Duration
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
	maxItems int64,
	maximumCost int64,
	cleanupFrequency time.Duration,
) CacheOptions {
	return CacheOptions{
		maxItems:         maxItems,
		maxCost:          maximumCost,
		cleanupFrequency: cleanupFrequency,
	}
}
