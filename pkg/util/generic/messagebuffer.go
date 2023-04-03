package generic

import (
	"fmt"
	"sync"
)

const (
	MessageBufferMaxSize = 1024 * 1024 // 1MB-ish
)

// sizable is an interface we expect the stored item to implement
// so that we know how large its data is before we agree to store
// it.
type sizable interface {
	GetDataSize() int64
}

type MessageBuffer[T sizable] struct {
	mu               sync.Mutex
	wait             *sync.Cond
	currentSizeBytes int64 // current buffer size
	maxSizeBytes     int64 // max buffer size size
	queue            []*T
	closed           bool
}

func NewMessageBuffer[T sizable](maxItems int64, maxSizeBytes int64) *MessageBuffer[T] {
	rb := &MessageBuffer[T]{
		queue:        make([]*T, 0, maxItems),
		maxSizeBytes: maxSizeBytes,
	}
	rb.wait = sync.NewCond(&rb.mu)
	return rb
}

func (rb *MessageBuffer[T]) Enqueue(msg *T) error {
	dataSize := (*msg).GetDataSize()

	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		return fmt.Errorf("buffer is already closed")
	}

	// Adding this T would exceed our preferred max size so drop the
	// message instead.
	if dataSize+rb.currentSizeBytes > rb.maxSizeBytes {
		rb.wait.Signal()
		return nil
	}

	rb.queue = append(rb.queue, msg)
	rb.currentSizeBytes += dataSize
	rb.wait.Signal()

	return nil
}

// Blocks waiting for a message if it cannot dequeue one
// immediately
func (rb *MessageBuffer[T]) Dequeue() (*T, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for len(rb.queue) == 0 && !rb.closed {
		// Wait for something to be enqueued
		rb.wait.Wait()
	}

	if rb.closed {
		return nil, fmt.Errorf("buffer closed")
	}

	msg := rb.queue[0]
	rb.queue = rb.queue[1:]
	rb.currentSizeBytes -= (*msg).GetDataSize()
	return msg, nil
}

func (rb *MessageBuffer[T]) Close() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		return
	}

	// Mark the buffer as closed and let everyone
	// know.
	rb.closed = true
	rb.wait.Broadcast()
}

func (rb *MessageBuffer[T]) Drain() []*T {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	remaining := make([]*T, 0, len(rb.queue))
	remaining = append(remaining, rb.queue...)
	rb.currentSizeBytes = 0
	return remaining
}
