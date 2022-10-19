package inprocess

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type InProcessTransportClusterConfig struct {
	Count int
	// a function to get the network latency for a given node id
	GetMessageDelay func(fromIndex, toIndex int) time.Duration
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
		cluster.transports = append(cluster.transports, transport)
	}

	for fromTransportIndex, fromTransport := range cluster.transports {
		// override the publish handler for each transport and write to all the other transports
		fromTransport.publishHandler = func(ctx context.Context, ev model.JobEvent) error {
			for toTransportIndex, toTransport := range cluster.transports {
				toTransport := toTransport
				fromTransportIndex := fromTransportIndex
				toTransportIndex := toTransportIndex
				go func() {
					if cluster.config.GetMessageDelay != nil {
						time.Sleep(cluster.config.GetMessageDelay(fromTransportIndex, toTransportIndex))
					}
					err := toTransport.applyEvent(ctx, ev)
					if err != nil {
						log.Error().Msgf("error in handle event: %s\n%+v", err, ev)
					}
				}()
			}
			return nil
		}
	}
	return cluster, nil
}

func (cluster *InProcessTransportCluster) GetTransport(i int) *InProcessTransport {
	return cluster.transports[i]
}
