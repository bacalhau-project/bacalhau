package pubsub

import (
	"context"
	"reflect"

	"github.com/rs/zerolog/log"
)

type ChainedPublisher[T any] struct {
	publishers   []Publisher[T]
	ignoreErrors bool
}

func NewChainedPublisher[T any](ignoreErrors bool) *ChainedPublisher[T] {
	return &ChainedPublisher[T]{
		ignoreErrors: ignoreErrors,
	}
}

// Add publisher to the chain
func (c *ChainedPublisher[T]) Add(publisher Publisher[T]) {
	c.publishers = append(c.publishers, publisher)
}

func (c *ChainedPublisher[T]) Publish(ctx context.Context, message T) error {
	for _, publisher := range c.publishers {
		err := publisher.Publish(ctx, message)
		if err != nil {
			if !c.ignoreErrors {
				return err
			} else {
				log.Ctx(ctx).Warn().Err(err).Msgf("error publishing message by %s", reflect.TypeOf(publisher))
			}
		}
	}
	return nil
}

// compile-time interface assertions
var _ Publisher[string] = (*ChainedPublisher[string])(nil)
