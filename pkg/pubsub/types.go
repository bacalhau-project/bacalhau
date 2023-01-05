package pubsub

import "context"

// PubSub enables publishing messages to subscribers
type PubSub[T any] interface {
	// Publish a message
	Publish(ctx context.Context, message T) error
	// Subscribe to messages
	Subscribe(ctx context.Context, subscriber Subscriber[T])
}

// Subscriber handles messages publishes to PubSub
type Subscriber[T any] interface {
	Handle(ctx context.Context, message T) error
}

// SubscriberFunc is a helper function that implements Subscriber interface
type SubscriberFunc[T any] func(ctx context.Context, message T) error

func (f SubscriberFunc[T]) Handle(ctx context.Context, message T) error {
	return f(ctx, message)
}
