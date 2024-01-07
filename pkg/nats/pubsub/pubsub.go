package pubsub

import (
	"context"
	"errors"
	"reflect"
	realsync "sync"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type PubSubParams struct {
	// Subject is the NATS subject to publish to. It is also used as the subscription subject if SubscriptionSubject is empty.
	Subject string
	// SubscriptionSubject is the NATS subject to subscribe to. If empty, Subject is used.
	// This is useful when the subscription subject is different from the publishing subject, e.g. when using wildcards.
	SubscriptionSubject string
	// Conn is the NATS connection to use for publishing and subscribing.
	Conn *nats.Conn
}
type PubSub[T any] struct {
	subject             string
	subscriptionSubject string
	conn                *nats.Conn

	subscription   *nats.Subscription
	subscriber     pubsub.Subscriber[T]
	subscriberOnce realsync.Once
	closeOnce      realsync.Once
}

func NewPubSub[T any](params PubSubParams) (*PubSub[T], error) {
	newPubSub := &PubSub[T]{
		conn:                params.Conn,
		subject:             params.Subject,
		subscriptionSubject: params.SubscriptionSubject,
	}
	if newPubSub.subscriptionSubject == "" {
		newPubSub.subscriptionSubject = newPubSub.subject
	}
	return newPubSub, nil
}

func (p *PubSub[T]) Publish(ctx context.Context, message T) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/pubsub/nats.publish")
	defer span.End()

	payload, err := marshaller.JSONMarshalWithMax(message)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Trace().Msgf("Sending message %+v", message)
	return p.conn.Publish(p.subject, payload)
}

func (p *PubSub[T]) Subscribe(ctx context.Context, subscriber pubsub.Subscriber[T]) (err error) {
	var firstSubscriber bool
	p.subscriberOnce.Do(func() {
		log.Ctx(ctx).Debug().Msgf("Subscribing to subject %s", p.subscriptionSubject)

		// register the subscriber
		p.subscriber = subscriber

		// subscribe to the subject
		p.subscription, err = p.conn.Subscribe(p.subscriptionSubject, func(msg *nats.Msg) {
			p.readMessage(context.Background(), msg)
		})
		if err != nil {
			return
		}

		firstSubscriber = true
	})
	if err != nil {
		return err
	}
	if !firstSubscriber {
		err = errors.New("only a single subscriber is allowed. Use ChainedSubscriber to chain multiple subscribers")
	}
	return err
}

func (p *PubSub[T]) readMessage(ctx context.Context, msg *nats.Msg) {
	var payload T
	err := marshaller.JSONUnmarshalWithMax(msg.Data, &payload)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error unmarshalling nats payload from subject %s", msg.Subject)
		return
	}

	err = p.subscriber.Handle(ctx, payload)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error in handle message of type: %s", reflect.TypeOf(payload))
	}
}

func (p *PubSub[T]) Close(ctx context.Context) (err error) {
	p.closeOnce.Do(func() {
		if p.subscription != nil {
			err = p.subscription.Unsubscribe()
		}
	})
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Msgf("done closing nats pubsub for subject %s", p.subscriptionSubject)
	return nil
}

// compile-time interface assertions
var _ pubsub.PubSub[string] = (*PubSub[string])(nil)
