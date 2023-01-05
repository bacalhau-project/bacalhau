package pubsub

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

// SingletonPubSub is an abstract pubsub that only allows one subscriber
type SingletonPubSub[T any] struct {
	Subscriber     Subscriber[T]
	subscriberOnce sync.Once
}

func (p *SingletonPubSub[T]) Subscribe(ctx context.Context, subscriber Subscriber[T]) {
	var firstSubscriber bool
	p.subscriberOnce.Do(func() {
		p.Subscriber = subscriber
		firstSubscriber = true
	})
	if !firstSubscriber {
		log.Warn().Msg("Only a single subscriber is allowed. Use ChainedSubscriber to chain multiple subscribers")
	}
}
