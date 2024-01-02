package pubsub

import "context"

// PubSub enables publishing messages to subscribers
type PubSub[T any] interface {
	// Publish a message
	Publish(ctx context.Context, message T) error
	// Subscribe to messages
	Subscribe(ctx context.Context, subscriber Subscriber[T]) error
	// Close the PubSub and release resources, if any
	Close(ctx context.Context) error
}

// Publisher handles messages publishes to PubSub
type Publisher[T any] interface {
	Publish(ctx context.Context, message T) error
}

// Subscriber handles messages publishes to PubSub
type Subscriber[T any] interface {
	Handle(ctx context.Context, message T) error
}

// PublisherFunc is a helper function that implements Publisher interface
type PublisherFunc[T any] func(ctx context.Context, message T) error

func (f PublisherFunc[T]) Publish(ctx context.Context, message T) error {
	return f(ctx, message)
}

// SubscriberFunc is a helper function that implements Subscriber interface
type SubscriberFunc[T any] func(ctx context.Context, message T) error

func (f SubscriberFunc[T]) Handle(ctx context.Context, message T) error {
	return f(ctx, message)
}
