package pkg

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/requester/pubsub/jobinfo"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

type ConsumerParams struct {
	Libp2pHost host.Host
	Datastore  *PostgresDatastore
}

// Consumer registers to the gossipsub topic and inserts the received job info into the datastore.
type Consumer struct {
	libp2pHost  host.Host
	pubSub      pubsub.PubSub[jobinfo.Envelope]
	cleanupFunc func(context.Context)
	datastore   *PostgresDatastore
}

func NewConsumer(params ConsumerParams) *Consumer {
	consumer := &Consumer{
		libp2pHost: params.Libp2pHost,
		datastore:  params.Datastore,
	}
	return consumer
}

func (c *Consumer) Start(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	gossipSub, err := libp2p_pubsub.NewGossipSub(ctx, c.libp2pHost)
	if err != nil {
		return err
	}

	// PubSub to read node info from the network
	log.Info().Str("Topic", node.JobInfoTopic).Msg("Subscribing")
	jobInfoPubSub, err := libp2p.NewPubSub[jobinfo.Envelope](libp2p.PubSubParams{
		Host:      c.libp2pHost,
		TopicName: node.JobInfoTopic,
		PubSub:    gossipSub,
	})
	if err != nil {
		return err
	}
	err = jobInfoPubSub.Subscribe(ctx, pubsub.SubscriberFunc[jobinfo.Envelope](c.datastore.InsertJobInfo))
	if err != nil {
		return err
	}

	c.cleanupFunc = func(ctx context.Context) {
		cleanupErr := jobInfoPubSub.Close(ctx)
		util.LogDebugIfContextCancelled(ctx, cleanupErr, "job info pubsub")
		cancel()
	}
	return nil
}

func (c *Consumer) Stop(ctx context.Context) error {
	if c.cleanupFunc != nil {
		c.cleanupFunc(ctx)
	}
	return nil
}
