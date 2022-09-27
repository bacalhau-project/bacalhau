package sync

import (
	"context"
	"sync"
)

// barrier represents a single barrier with multiple waiters.
type barrier struct {
	sync.Mutex
	count int
	zcs   []*zeroCounter
}

// wait waits for the barrier to reach a certain target.
func (b *barrier) wait(ctx context.Context, target int) error {
	b.Lock()

	// If we're already over the target, return immediately.
	if target <= b.count {
		b.Unlock()
		return nil
	}

	// Create a zero counter to wait for target - count elements to signal entry.
	// It also returns if the context fires.
	zc := newZeroCounter(ctx, target-b.count)

	// Store the zero counter, unlock the barrier and wait for it to be reached.
	b.zcs = append(b.zcs, zc)
	b.Unlock()
	return zc.wait()
}

// inc increments the barrier by one unit. To do so, we increment
// the counter and tell all the channels we received a new entry.
func (b *barrier) inc() int {
	b.Lock()
	defer b.Unlock()

	b.count += 1
	count := b.count

	for _, zc := range b.zcs {
		zc.dec()
	}

	return count
}

// isDone returns true if all the counters for this barrier have reached zero.
func (b *barrier) isDone() bool {
	b.Lock()
	defer b.Unlock()

	for _, zc := range b.zcs {
		if !zc.done() {
			return false
		}
	}

	return true
}

type zeroCounter struct {
	sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	count  int
}

func newZeroCounter(ctx context.Context, target int) *zeroCounter {
	ctx, cancel := context.WithCancel(ctx)

	return &zeroCounter{
		count:  target,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (w *zeroCounter) dec() {
	w.Lock()
	defer w.Unlock()

	if w.count <= 0 {
		return
	}

	w.count -= 1
	if w.count <= 0 {
		w.cancel()
	}
}

func (w *zeroCounter) wait() error {
	<-w.ctx.Done()

	// If the counter is done, i.e., if it
	// reached 0 or lower, we do not return
	// an error.
	if w.done() {
		return nil
	}

	return w.ctx.Err()
}

func (w *zeroCounter) done() bool {
	return w.count <= 0
}
