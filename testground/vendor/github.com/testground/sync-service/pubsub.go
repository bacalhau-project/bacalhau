package sync

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type subscription struct {
	sync.Mutex
	ctx    context.Context
	outCh  chan string
	doneCh chan error
	last   int
	closed bool
}

func (s *subscription) write(d time.Duration, msg string) error {
	s.Lock()
	defer s.Unlock()

	if s.closed {
		return fmt.Errorf("subscription already closed")
	}

	ctx, cancel := context.WithTimeout(s.ctx, d)
	defer cancel()

	select {
	case s.outCh <- msg:
		// Increment last for the next ID.
		s.last++
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *subscription) close() {
	s.Lock()
	defer s.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	s.doneCh <- s.ctx.Err()
	close(s.doneCh)
	close(s.outCh)
}

type pubsub struct {
	// ctx includes the context of the caller's context, wrapped
	// by cancelation through cancel.
	ctx    context.Context
	cancel context.CancelFunc

	// lastmod tracks the last time this pubsub instance was accessed
	// i.e., the last time something, or someone, subscribed or published.
	lastmod time.Time

	// wg tracks if the worker for this pubsub instance is done or not.
	// The worker stops when the context is canceled.
	wg sync.WaitGroup

	// msgs tracks all the messages published to this pubsub instance.
	// It is needed so when new subscriptions are created, we can send
	// all messages retroactively, in order.
	msgs   []string
	msgsMu sync.RWMutex

	// subs tracks the subscriptions for this pubsub instance.
	subs   []*subscription
	subsMu sync.RWMutex
}

func (ps *pubsub) subscribe(ctx context.Context) *subscription {
	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	sub := &subscription{
		ctx:    ctx,
		closed: false,
		outCh:  make(chan string),
		doneCh: make(chan error, 1),
		last:   0,
	}

	ps.lastmod = time.Now()
	ps.subs = append(ps.subs, sub)
	return sub
}

func (ps *pubsub) publish(msg string) int {
	ps.msgsMu.Lock()
	defer ps.msgsMu.Unlock()

	ps.lastmod = time.Now()
	ps.msgs = append(ps.msgs, msg)
	return len(ps.msgs)
}

func (ps *pubsub) worker() {
	ticker := time.NewTicker(100 * time.Millisecond)
	ps.wg.Add(1)
	defer ps.wg.Done()

	for {
		select {
		case <-ticker.C:
			ps.writeMessages()
		case <-ps.ctx.Done():
			ps.cancel()
			return
		}
	}
}

func (ps *pubsub) writeMessages() {
	ps.subsMu.RLock()
	defer ps.subsMu.RUnlock()

	for _, sub := range ps.subs {
		func() {
			ps.msgsMu.RLock()
			defer ps.msgsMu.RUnlock()

			for _, msg := range ps.msgs[sub.last:] {
				err := sub.write(time.Second*5, msg)
				if err != nil {
					log.Warnf("cannot send message to subscriber: %w", err)
					break
				}
			}
		}()
	}
}

func (ps *pubsub) isDone() bool {
	ps.subsMu.RLock()
	defer ps.subsMu.RUnlock()

	for _, sub := range ps.subs {
		if sub.ctx.Err() == nil {
			return false
		} else if !sub.closed {
			sub.close()
		}
	}

	return true
}

func (ps *pubsub) close() {
	ps.cancel()

	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	for _, sub := range ps.subs {
		sub.close()
	}

	ps.wg.Wait()
}
