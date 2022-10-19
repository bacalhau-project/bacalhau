package inprocess

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

type InProcessTransportClusterConfig struct {
	Count int
	// a function to get the network latency for a given node id
	GetMessageDelay func(index int) time.Duration
}

// this is a set of "connected" in process transports
// meaning that an event written to one of them will be delivered
// to the subscribeFunctions and seen events of all of them
type InProcessTransportCluster struct {
	transports []*InProcessTransport
	config     InProcessTransportClusterConfig
}

func NewInProcessTransportCluster(config InProcessTransportClusterConfig) (*InProcessTransportCluster, error) {
	cluster := &InProcessTransportCluster{
		transports: []*InProcessTransport{},
		config:     config,
	}
	for i := 0; i < config.Count; i++ {
		transport, err := NewInprocessTransport()
		if err != nil {
			return nil, err
		}
		transport.publishHandler = cluster.Publish
		cluster.transports = append(cluster.transports, transport)
	}
	return cluster, nil
}

func (cluster *InProcessTransportCluster) Start(ctx context.Context) error {
	return nil
}

func (cluster *InProcessTransportCluster) Shutdown(ctx context.Context) error {
	return nil
}

func (cluster *InProcessTransportCluster) HostID() string {
	return "cluster_root"
}

func (cluster *InProcessTransportCluster) HostAddrs() ([]multiaddr.Multiaddr, error) {
	return []multiaddr.Multiaddr{}, nil
}

func (cluster *InProcessTransportCluster) GetEvents() []model.JobEvent {
	return []model.JobEvent{}
}

/*

  pub / sub

*/

// this function is assigned to the "publishHandler" of each transport
// this means that calling "Publish" on one of the nodes transports
// will end up here where we distribute the event to all the other nodes at the same time
func (cluster *InProcessTransportCluster) Publish(ctx context.Context, ev model.JobEvent) error {
	for i, transport := range cluster.transports {
		transport := transport
		i := i
		go func() {
			if cluster.config.GetMessageDelay != nil {
				time.Sleep(cluster.config.GetMessageDelay(i))
			}
			err := transport.applyEvent(ctx, ev)
			if err != nil {
				log.Error().Msgf("error in handle event: %s\n%+v", err, ev)
			}
		}()
	}
	return nil
}

func (cluster *InProcessTransportCluster) Subscribe(ctx context.Context, fn transport.SubscribeFn) {}

/*
encrypt / decrypt
*/

func (*InProcessTransportCluster) Encrypt(ctx context.Context, data, encryptionKeyBytes []byte) ([]byte, error) {
	return data, nil
}

func (*InProcessTransportCluster) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	return data, nil
}

func (cluster *InProcessTransportCluster) GetTransport(i int) *InProcessTransport {
	return cluster.transports[i]
}

// Static check to ensure that InProcessTransportCluster implements Transport:
var _ transport.Transport = (*InProcessTransportCluster)(nil)
