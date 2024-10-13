package generic

import (
	"container/ring"
	"sync"
)

const DefaultRingBufferSize = 16384

type RingBuffer[T any] struct {
	writeR     *ring.Ring
	readR      *ring.Ring
	mu         sync.Mutex
	ready      *sync.Cond
	readCount  int64
	wroteCount int64
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
	if size == 0 {
		size = DefaultRingBufferSize
	}

	r := ring.New(size)
	rb := &RingBuffer[T]{
		readR:  r,
		writeR: r,
	}
	rb.ready = sync.NewCond(&rb.mu)
	return rb
}

// Enqueue will continue to write data to the ring buffer even if nothing
// is reading from it.
func (r *RingBuffer[T]) Enqueue(data T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.writeR.Value = data
	r.writeR = r.writeR.Next()
	r.wroteCount += 1
	r.ready.Signal()
}

func (r *RingBuffer[T]) Dequeue() T {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If the current cell is nil then wait until something is written
	if r.readR.Value == nil {
		r.ready.Wait()
	}

	data := r.readR.Value
	r.readR = r.readR.Next()
	r.readCount += 1

	if data == nil {
		return *new(T)
	}

	return data.(T)
}

func (r *RingBuffer[T]) Drain() []T {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := r.wroteCount - r.readCount
	if count <= 0 {
		return nil
	}

	ret := make([]T, count)
	for i := int64(0); i < count; i++ {
		if r.readR.Value == nil {
			// If we enqueued nil for some reason, make sure
			// we skip it when draining the items.
			continue
		}

		ret[i] = r.readR.Value.(T)
		r.readR = r.readR.Next()
	}
	return ret
}

func (r *RingBuffer[T]) Each(f func(any)) {
	r.readR.Do(f)
}
