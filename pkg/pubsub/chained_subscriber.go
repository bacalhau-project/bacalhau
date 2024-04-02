package pubsub

import (
	"context"
	"reflect"

	"github.com/rs/zerolog/log"
)

type ChainedSubscriber[T any] struct {
	subscribers  []Subscriber[T]
	ignoreErrors bool
}

func NewChainedSubscriber[T any](ignoreErrors bool) *ChainedSubscriber[T] {
	return &ChainedSubscriber[T]{
		ignoreErrors: ignoreErrors,
	}
}

// Add subscriber to the chain
func (c *ChainedSubscriber[T]) Add(subscriber Subscriber[T]) {
	c.subscribers = append(c.subscribers, subscriber)
}

// Handle message by calling all subscribers in the chain
func (c *ChainedSubscriber[T]) Handle(ctx context.Context, message T) error {
	for _, subscriber := range c.subscribers {
		err := subscriber.Handle(ctx, message)
		if err != nil {
			if !c.ignoreErrors {
				return err
			} else {
				log.Ctx(ctx).Warn().Err(err).Msgf("error handling message by %s", reflect.TypeOf(subscriber))
			}
		}
	}
	return nil
}

// compile-time interface assertions
var _ Subscriber[string] = (*ChainedSubscriber[string])(nil)
