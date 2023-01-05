package libp2p

import (
	"context"
	"reflect"

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
	pubsub.SingletonPubSub[T]
	hostID       string
	topic        *libp2p_pubsub.Topic
	subscription *libp2p_pubsub.Subscription
	ignoreLocal  bool
}

func NewPubSub[T any](ctx context.Context, params PubSubParams) (*PubSub[T], error) {
	topic, err := params.PubSub.Join(params.TopicName)
	if err != nil {
		return nil, err
	}

	subscription, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	newPubSub := &PubSub[T]{
		hostID:       params.Host.ID().String(),
		topic:        topic,
		subscription: subscription,
		ignoreLocal:  params.IgnoreLocal,
	}

	go newPubSub.listenForEvents(ctx)
	return newPubSub, nil
}

func (p *PubSub[T]) Publish(ctx context.Context, message T) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/pubsub/libp2p.Publish")
	defer span.End()

	payload, err := model.JSONMarshalWithMax(message)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Trace().Msgf("Sending message %+v", payload)
	return p.topic.Publish(ctx, payload)
}

func (p *PubSub[T]) listenForEvents(ctx context.Context) {
	for {
		msg, err := p.subscription.Next(ctx)
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
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
	ctx = logger.ContextWithNodeIDLogger(ctx, p.hostID)
	var payload T
	err := model.JSONUnmarshalWithMax(msg.Data, &payload)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error unmarshalling libp2p payload: %v", err)
		return
	}

	err = p.Subscriber.Handle(ctx, payload)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error in handle message of type: %s", reflect.TypeOf(payload))
	}
}

// compile-time interface assertions
var _ pubsub.PubSub[string] = (*PubSub[string])(nil)
