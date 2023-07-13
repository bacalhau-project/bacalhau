package pubsub

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggingSubscriber that just logs the message.
type LoggingSubscriber[T any] struct {
	level zerolog.Level
}

func NewLoggingSubscriber[T any](level zerolog.Level) *LoggingSubscriber[T] {
	return &LoggingSubscriber[T]{
		level: level,
	}
}

func (c *LoggingSubscriber[T]) Handle(ctx context.Context, message T) error {
	log.Ctx(ctx).WithLevel(c.level).Msgf("%+v", message)
	return nil
}

// compile-time interface assertions
var _ Subscriber[string] = (*LoggingSubscriber[string])(nil)
