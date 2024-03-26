package heartbeat

import (
	"context"

	natsPubSub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type HeartbeatServer struct {
	subscription *natsPubSub.PubSub[Heartbeat]
}

func NewServer(conn *nats.Conn) (*HeartbeatServer, error) {
	subParams := natsPubSub.PubSubParams{
		Subject: heartbeatTopic,
		Conn:    conn,
	}

	subscription, err := natsPubSub.NewPubSub[Heartbeat](subParams)
	if err != nil {
		return nil, err
	}

	return &HeartbeatServer{subscription: subscription}, nil
}

func (h *HeartbeatServer) Start(ctx context.Context) error {
	if err := h.subscription.Subscribe(ctx, h); err != nil {
		return err
	}

	go func(ctx context.Context) {
		log.Ctx(ctx).Info().Msg("Heartbeat server started")
		<-ctx.Done()
		_ = h.subscription.Close(ctx)
		log.Ctx(ctx).Info().Msg("Heartbeat server shutdown")
	}(ctx)

	return nil
}

func (h *HeartbeatServer) Handle(ctx context.Context, message Heartbeat) error {
	log.Ctx(ctx).Trace().Msgf("heartbeat received from %s", message.NodeID)

	// TODO: Process the heartbeat (e.g. update the node's last seen time)

	return nil
}

var _ pubsub.Subscriber[Heartbeat] = (*HeartbeatServer)(nil)
