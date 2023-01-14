package pubsub

import (
	"context"
	"errors"
	"sync"
)

// InMemoryPubSub is a simple in-memory pubsub implementation used for testing
type InMemoryPubSub[T any] struct {
	subscriber     Subscriber[T]
	subscriberOnce sync.Once
}

func NewInMemoryPubSub[T any]() *InMemoryPubSub[T] {
	return &InMemoryPubSub[T]{}
}

func (p *InMemoryPubSub[T]) Publish(ctx context.Context, message T) error {
	if p.subscriber != nil {
		return p.subscriber.Handle(ctx, message)
	}
	return nil
}

func (p *InMemoryPubSub[T]) Subscribe(ctx context.Context, subscriber Subscriber[T]) error {
	var firstSubscriber bool
	p.subscriberOnce.Do(func() {
		p.subscriber = subscriber
		firstSubscriber = true
	})
	if !firstSubscriber {
		return errors.New("only a single subscriber is allowed. Use ChainedSubscriber to chain multiple subscribers")
	}
	return nil
}

func (p *InMemoryPubSub[T]) Close(ctx context.Context) error {
	return nil
}

// InMemorySubscriber is a simple in-memory subscriber implementation used for testing
type InMemorySubscriber[T any] struct {
	events        []T
	badSubscriber bool
}

func NewInMemorySubscriber[T any]() *InMemorySubscriber[T] {
	return &InMemorySubscriber[T]{
		events: make([]T, 0),
	}
}

func (s *InMemorySubscriber[T]) Handle(ctx context.Context, message T) error {
	if s.badSubscriber {
		return errors.New("failed to handler message as I am a bad subscriber")
	}
	s.events = append(s.events, message)
	return nil
}

func (s *InMemorySubscriber[T]) Events() []T {
	res := s.events
	s.events = make([]T, 0)
	return res
}

// compile-time interface assertions
var _ PubSub[string] = (*InMemoryPubSub[string])(nil)
var _ Subscriber[string] = (*InMemorySubscriber[string])(nil)
