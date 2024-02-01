package basic

import (
	"time"
)

type Config struct {
	maxCost          uint64
	cleanupFrequency time.Duration
	evictionFunction EvictItemFunc
	defaultTTL       int64
}

// EvictItemFunc can be used to tell the cache not to evict an
// item based on the provided, key, value or expiryTime.
type EvictItemFunc func(key string, cost uint64, expiryTime int64, now int64) bool

type Option func(*Config)

func WithMaxCost(maxCost uint64) Option {
	return func(o *Config) {
		o.maxCost = maxCost
	}
}

func WithCleanupFrequency(cleanupFrequency time.Duration) Option {
	return func(o *Config) {
		o.cleanupFrequency = cleanupFrequency
	}
}

func WithEvictionFunction(f EvictItemFunc) Option {
	return func(o *Config) {
		o.evictionFunction = f
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(o *Config) {
		o.defaultTTL = int64(ttl.Seconds())
	}
}
