package pubsub

import (
	"context"
	"errors"
)

// InMemoryPubSub is a simple in-memory pubsub implementation used for testing
type InMemoryPubSub[T any] struct {
	SingletonPubSub[T]
}

func NewInMemoryPubSub[T any]() *InMemoryPubSub[T] {
	return &InMemoryPubSub[T]{}
}

func (p *InMemoryPubSub[T]) Publish(ctx context.Context, message T) error {
	return p.Subscriber.Handle(ctx, message)
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
