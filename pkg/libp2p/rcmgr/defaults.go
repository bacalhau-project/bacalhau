package rcmgr

import (
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	"github.com/libp2p/go-libp2p"
	libp2p_rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/host/resource-manager/obs"
)

func SetDefaultServiceLimits(config *libp2p_rcmgr.ScalingLimitConfig) {
	// Requester -> Compute nodes
	// reasoning behind these limits:
	// - Requester nodes should have a high number of outbound streams to compute nodes
	// - Compute nodes should have a high number of inbound streams from requester nodes
	// - Since there are few requester nodes in the network, we should set a high limit for the number of streams per peer
	config.AddServiceLimit(
		bprotocol.ComputeServiceName,
		libp2p_rcmgr.BaseLimit{StreamsInbound: 1024, StreamsOutbound: 4096, Streams: 4096},
		libp2p_rcmgr.BaseLimitIncrease{StreamsInbound: 512, StreamsOutbound: 2048, Streams: 2048},
	)
	config.AddServicePeerLimit(
		bprotocol.ComputeServiceName,
		libp2p_rcmgr.BaseLimit{StreamsInbound: 1024, StreamsOutbound: 1024, Streams: 1024},
		libp2p_rcmgr.BaseLimitIncrease{StreamsInbound: 128, StreamsOutbound: 128, Streams: 128},
	)

	// Compute -> Requester nodes
	// reasoning behind these limits:
	// - Compute nodes should have a high number of outbound streams to requester nodes
	// - Requester nodes should have a high number of inbound streams from compute nodes
	// - Since there are few requester nodes in the network, we should set a high limit for the number of streams per peer
	config.AddServiceLimit(
		bprotocol.CallbackServiceName,
		libp2p_rcmgr.BaseLimit{StreamsInbound: 4096, StreamsOutbound: 1024, Streams: 4096},
		libp2p_rcmgr.BaseLimitIncrease{StreamsInbound: 2048, StreamsOutbound: 512, Streams: 2048},
	)
	config.AddServicePeerLimit(
		bprotocol.CallbackServiceName,
		libp2p_rcmgr.BaseLimit{StreamsInbound: 1024, StreamsOutbound: 1024, Streams: 1024},
		libp2p_rcmgr.BaseLimitIncrease{StreamsInbound: 128, StreamsOutbound: 128, Streams: 128},
	)
}

var DefaultResourceManager = func(cfg *libp2p.Config) error {
	// Default memory limit: 1/8th of total memory, minimum 128MB, maximum 1GB
	limits := libp2p_rcmgr.DefaultLimits
	libp2p.SetDefaultServiceLimits(&limits)
	SetDefaultServiceLimits(&limits)

	str, err := obs.NewStatsTraceReporter()
	if err != nil {
		return err
	}
	mgr, err := libp2p_rcmgr.NewResourceManager(
		libp2p_rcmgr.NewFixedLimiter(limits.AutoScale()),
		libp2p_rcmgr.WithTraceReporter(str),
	)
	if err != nil {
		return err
	}

	return cfg.Apply(libp2p.ResourceManager(mgr))
}
