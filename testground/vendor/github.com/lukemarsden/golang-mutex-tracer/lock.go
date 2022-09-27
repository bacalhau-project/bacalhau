package muxtracer

import (
	"sync"
	"sync/atomic"
)

type Mutex struct {
	lock sync.Mutex

	// internal trace fields
	threshold        uint64 // 0 when disabled, else threshold in nanoseconds
	beginAwaitLock   uint64 // start time in unix nanoseconds from start waiting for lock
	beginAwaitUnlock uint64 // start time in unix nanoseconds from start waiting for unlock
	lockObtained     uint64 // once we've entered the lock in unix nanoseconds
	id               []byte // if set this will be printed as string
}

func (m *Mutex) Lock() {
	tracingThreshold := m.isTracing()
	if tracingThreshold != 0 {
		m.traceBeginAwaitLock()
	}

	// actual lock
	m.lock.Lock()

	if tracingThreshold != 0 {
		m.traceEndAwaitLock(tracingThreshold)
	}
}

func (m *Mutex) Unlock() {
	tracingThreshold := m.isTracing()
	if tracingThreshold != 0 {
		m.traceBeginAwaitUnlock()
	}

	// unlock
	m.lock.Unlock()

	if tracingThreshold != 0 {
		m.traceEndAwaitUnlock(tracingThreshold)
	}
}

func (m *Mutex) isTracing() Threshold {
	tracingThreshold := atomic.LoadUint64(&m.threshold)
	if tracingThreshold == 0 {
		// always on?
		tracingThreshold = atomic.LoadUint64(&defaultThreshold)
	}
	return Threshold(tracingThreshold)
}

func (m *Mutex) traceBeginAwaitLock() {
	atomic.StoreUint64(&m.beginAwaitLock, now())
}

func (m *Mutex) traceEndAwaitLock(threshold Threshold) {
	ts := now() // first obtain the current time
	start := atomic.LoadUint64(&m.beginAwaitLock)
	atomic.StoreUint64(&m.lockObtained, uint64(ts))
	var took uint64
	if start < ts {
		// check for no overflow
		took = ts - start
	}
	if took >= uint64(threshold) {
		logViolation(Id(m.id), threshold, Actual(took), Now(ts), ViolationLock)
	}
}

func (m *Mutex) traceBeginAwaitUnlock() {
	atomic.StoreUint64(&m.beginAwaitUnlock, now())
}

func (m *Mutex) traceEndAwaitUnlock(threshold Threshold) {
	ts := now() // first obtain the current time

	// lock obtained time (critical section)
	lockObtained := atomic.LoadUint64(&m.lockObtained)
	var took uint64
	if lockObtained < ts {
		// check for no overflow
		took = ts - lockObtained
	}

	if took >= uint64(threshold) && lockObtained > 0 {
		// lockObtained = 0 when the tracer is enabled half way
		logViolation(Id(m.id), threshold, Actual(took), Now(ts), ViolationCritical)
	}
}
