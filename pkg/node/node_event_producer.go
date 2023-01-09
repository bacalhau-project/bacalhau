package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

func getPeerIDS(transport transport.Transport) map[string][]peer.ID {
	defaultData := map[string][]peer.ID{}
	switch apiTransport := transport.(type) { //nolint:gocritic
	case *libp2p.LibP2PTransport:
		peers, err := apiTransport.GetPeers(context.Background())
		if err != nil {
			log.Error().Msgf("Error getting peers: %s", err.Error())
			return defaultData
		}
		return peers
	}
	return defaultData
}

func getCurrentCapacity(capacityTracker capacity.Tracker) (model.ResourceUsageData, model.ResourceUsageData) {
	total := capacityTracker.TotalCapacity(context.Background())
	available := capacityTracker.AvailableCapacity(context.Background())
	return total, available
}

func getDebugInfo(debugInfoProviders []model.DebugInfoProvider) ([]model.DebugInfo, error) {
	data := []model.DebugInfo{}
	for _, provider := range debugInfoProviders {
		debugInfo, err := provider.GetDebugInfo()
		if err != nil {
			return nil, err
		}
		data = append(data, debugInfo)
	}
	return data, nil
}

func getNodeEvent(
	transport transport.Transport,
	capacityTracker capacity.Tracker,
	debugInfoProviders []model.DebugInfoProvider,
) (model.NodeEvent, error) {
	id := transport.HostID()
	peerIDS := getPeerIDS(transport)
	total, available := getCurrentCapacity(capacityTracker)
	debugInfo, err := getDebugInfo(debugInfoProviders)
	if err != nil {
		return model.NodeEvent{}, err
	}
	return model.NodeEvent{
		EventTime:         time.Now(),
		NodeID:            id,
		EventName:         model.NodeEventAnnounce,
		TotalCapacity:     total,
		AvailableCapacity: available,
		Peers:             peerIDS,
		DebugInfo:         debugInfo,
	}, nil
}

func publishEventHandler(
	transport transport.Transport,
	capacityTracker capacity.Tracker,
	debugInfoProviders []model.DebugInfoProvider,
	publisher *eventhandler.ChainedNodeEventHandler,
) {
	event, err := getNodeEvent(transport, capacityTracker, debugInfoProviders)
	if err != nil {
		log.Error().Msgf("Error getting node event: %s", err.Error())
		return
	}
	err = publisher.HandleNodeEvent(context.Background(), event)
	if err != nil {
		log.Error().Msgf("Error handling node event: %s", err.Error())
	}
}

// this is a loop that will write a node event every X seconds
// this is so the network knows the node exists
// and what it's features are and show it's connected to
func nodeEventProducer(
	ctx context.Context,
	transport transport.Transport,
	capacityTracker capacity.Tracker,
	debugInfoProviders []model.DebugInfoProvider,
	publisher *eventhandler.ChainedNodeEventHandler,
	publishInterval time.Duration,
) {
	ticker := time.NewTicker(publishInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				publishEventHandler(transport, capacityTracker, debugInfoProviders, publisher)
			}
		}
	}()
}
