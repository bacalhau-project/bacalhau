package muxtracer

import (
	"sync"
	"sync/atomic"
	"time"
)

var defaultGlobalOpts Opts
var defaultGlobalOptsMux sync.RWMutex
var defaultThreshold uint64

func obtainGlobalOpts() Opts {
	defaultGlobalOptsMux.RLock()
	c := defaultGlobalOpts
	defaultGlobalOptsMux.RUnlock()
	return c
}

func init() {
	ResetDefaults()
}

func SetGlobalOpts(o Opts) {
	if o.Threshold < 0 {
		panic("threshold can not be negative")
	}
	defaultGlobalOptsMux.Lock()
	defaultGlobalOpts = o
	atomic.StoreUint64(&defaultThreshold, uint64(o.Threshold))
	defaultGlobalOptsMux.Unlock()
}

func ResetDefaults() {
	// default global opts
	o := Opts{
		Threshold: 100 * time.Millisecond,
		Enabled:   false, // by default needs toggle per lock
	}
	SetGlobalOpts(o)
}
