package libp2p

import (
	"context"
	"errors"
	"reflect"
	realsync "sync"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	"github.com/filecoin-project/bacalhau/pkg/system"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

type PubSubParams struct {
	Host        host.Host
	TopicName   string
	PubSub      *libp2p_pubsub.PubSub
	IgnoreLocal bool
}
type PubSub[T any] struct {
	hostID      string
	topicName   string
	pubSub      *libp2p_pubsub.PubSub
	ignoreLocal bool

	topic        *libp2p_pubsub.Topic
	subscription *libp2p_pubsub.Subscription

	subscriber     pubsub.Subscriber[T]
	subscriberOnce realsync.Once
	closeOnce      realsync.Once
}

func NewPubSub[T any](params PubSubParams) (*PubSub[T], error) {
	topic, err := params.PubSub.Join(params.TopicName)
	if err != nil {
		return nil, err
	}
	newPubSub := &PubSub[T]{
		hostID:      params.Host.ID().String(),
		pubSub:      params.PubSub,
		topic:       topic,
		topicName:   params.TopicName,
		ignoreLocal: params.IgnoreLocal,
	}
	return newPubSub, nil
}

func (p *PubSub[T]) Publish(ctx context.Context, message T) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/pubsub/libp2p.Publish.Publish")
	defer span.End()

	payload, err := model.JSONMarshalWithMax(message)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Trace().Msgf("Sending message %+v", message)
	return p.topic.Publish(ctx, payload)
}

func (p *PubSub[T]) Subscribe(ctx context.Context, subscriber pubsub.Subscriber[T]) (err error) {
	var firstSubscriber bool
	p.subscriberOnce.Do(func() {
		// register the subscriber
		p.subscriber = subscriber

		p.subscription, err = p.topic.Subscribe()
		if err != nil {
			return
		}

		// start listening for events
		go p.listenForEvents()
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

func (p *PubSub[T]) listenForEvents() {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), p.hostID)
	for {
		msg, err := p.subscription.Next(ctx)
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded || err == libp2p_pubsub.ErrSubscriptionCancelled {
				log.Ctx(ctx).Trace().Msgf("libp2p pubsub shutting down: %v", err)
			} else {
				log.Ctx(ctx).Error().Msgf(
					"libp2p encountered an unexpected error, shutting down: %v", err)
			}
			return
		}
		if p.ignoreLocal && msg.GetFrom().String() == p.hostID {
			continue
		}
		p.readMessage(ctx, msg)
	}
}

func (p *PubSub[T]) readMessage(ctx context.Context, msg *libp2p_pubsub.Message) {
	// TODO: we would enforce the claims to SourceNodeID here
	// i.e. msg.ReceivedFrom() should match msg.Data.JobEvent.SourceNodeID
	var payload T
	err := model.JSONUnmarshalWithMax(msg.Data, &payload)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error unmarshalling libp2p payload: %v", err)
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
			p.subscription.Cancel()
		}
		if p.topic != nil {
			err = p.topic.Close()
		}
	})
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Msgf("done closing libp2p pubsub for topic %s", p.topicName)
	return nil
}

// compile-time interface assertions
var _ pubsub.PubSub[string] = (*PubSub[string])(nil)
