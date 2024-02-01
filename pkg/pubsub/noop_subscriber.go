package pubsub

import (
	"context"
)

// NoopSubscriber is a subscriber that does nothing.
// It is useful for adding subscribers to the pubsub network that only participate in the gossip
// protocol, but do not actually handle messages. This is useful when the network is small and not enough peers
// are available to strengthen the network.
type NoopSubscriber[T any] struct {
}

func NewNoopSubscriber[T any]() *NoopSubscriber[T] {
	return &NoopSubscriber[T]{}
}

func (c *NoopSubscriber[T]) Handle(ctx context.Context, message T) error {
	return nil
}

// compile-time interface assertions
var _ Subscriber[string] = (*NoopSubscriber[string])(nil)
