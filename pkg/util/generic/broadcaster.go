package generic

import (
	"fmt"
	"sync"
	"time"
)

const (
	SafeWriteTimeout            = time.Duration(100) * time.Millisecond
	DefaultBroadcastChannelSize = 8
)

var errBroadcasterClosed error = fmt.Errorf("broadcaster is closed")

type Broadcaster[T any] struct {
	mu         sync.RWMutex
	clients    map[chan T]struct{}
	bufferSize int
	closed     bool
	autoClose  bool
}

func NewBroadcaster[T any](bufferSize int) *Broadcaster[T] {
	if bufferSize == 0 {
		bufferSize = DefaultBroadcastChannelSize
	}

	return &Broadcaster[T]{
		clients:    make(map[chan T]struct{}),
		closed:     false,
		bufferSize: bufferSize,
	}
}

func (b *Broadcaster[T]) SetAutoclose(val bool) {
	b.autoClose = val
}

func (b *Broadcaster[T]) IsClosed() bool {
	return b.closed
}

// Subscribe allows a client to request future entries and
// it does this by listening to the channel it is provided.
func (b *Broadcaster[T]) Subscribe() (chan T, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, errBroadcasterClosed
	}

	s := make(chan T, b.bufferSize)
	b.clients[s] = struct{}{}

	return s, nil
}

// Unsubscribe will remove the provided channel (which should have
// come from a previous call to Subscribe)
func (b *Broadcaster[T]) Unsubscribe(ch chan T) {
	defer b.mu.Unlock()
	b.mu.Lock()

	delete(b.clients, ch)

	if b.autoClose && len(b.clients) == 0 {
		b.closed = true
	}
}

// Broadcast will send the provided T to all currently subscribed
// channels as long as the channel is not full (which would cause
// the broadcaster to block until there was space on that channel).
func (b *Broadcaster[T]) Broadcast(data T) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return errBroadcasterClosed
	}

	if len(b.clients) == 0 {
		return nil
	}

	toRemove := make([]chan T, 0)

	for ch := range b.clients {
		// Attempt to write to the channel, if closed then add
		// it to the list of channels to auto unsubscribe
		isClosed := b.safeChannelSend(ch, data)
		if isClosed {
			toRemove = append(toRemove, ch)
		}
	}

	for _, ch := range toRemove {
		// Temporarily release the lock for the unsubscribe
		b.mu.Unlock()
		b.Unsubscribe(ch)
		b.mu.Lock()
	}

	return nil
}

// safeChannelSend will attempt to send the `value` to the provided channel `ch`,
// and ensuring that if the channel was previously closed, the panic is handled.
// If the channel is closed, this function will return true.
func (b *Broadcaster[T]) safeChannelSend(ch chan T, value T) (isClosed bool) {
	defer func() {
		if recover() != nil {
			isClosed = true
		}
	}()

	if len(ch) == b.bufferSize {
		// Full
		return
	}

	timer := time.NewTimer(SafeWriteTimeout)

	select {
	case ch <- value:
		return
	case <-timer.C:
		isClosed = true
		return
	}
}

func (b *Broadcaster[T]) Close() {
	b.closed = true
	for ch := range b.clients {
		close(ch)
	}
}
