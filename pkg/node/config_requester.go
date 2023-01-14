package node

import (
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type RequesterConfigParams struct {
	// Timeout config
	JobNegotiationTimeout      time.Duration
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration

	StateManagerBackgroundTaskInterval time.Duration
	NodeRankRandomnessRange            int
	NodeInfoStoreTTL                   time.Duration
	DiscoveredPeerStoreTTL             time.Duration
	SimulatorConfig                    model.SimulatorConfigRequester
}

type RequesterConfig struct {
	// Timeout config
	// JobNegotiationTimeout timeout value waiting for enough bids to be submitted for a job
	JobNegotiationTimeout time.Duration
	// MinJobExecutionTimeout requester will replace any job execution timeout that is less than this
	// value with DefaultJobExecutionTimeout.
	MinJobExecutionTimeout time.Duration
	// DefaultJobExecutionTimeout default value for running, verifying and publishing job results,
	// if the user didn't define one in the spec
	DefaultJobExecutionTimeout time.Duration

	// StateManagerBackgroundTaskInterval background task interval that periodically checks for
	// expired states among other things.
	StateManagerBackgroundTaskInterval time.Duration
	// NodeRankRandomnessRange defines the range of randomness used to rank nodes
	NodeRankRandomnessRange int
	// NodeInfoStoreTTL defines how long a node info is kept in the store
	NodeInfoStoreTTL time.Duration
	// DiscoveredPeerStoreTTL defines how long a peer is kept in the libp2p host's peerstore so that it can be connected to after the node was
	// discovered outside of the libp2p host's peerstore.
	// We only need to store the peer long enough for the requester to connect to the compute node for the duration of the job.
	DiscoveredPeerStoreTTL time.Duration
	SimulatorConfig        model.SimulatorConfigRequester
}

func NewRequesterConfigWithDefaults() RequesterConfig {
	return NewRequesterConfigWith(DefaultRequesterConfig)
}

//nolint:gosimple
func NewRequesterConfigWith(params RequesterConfigParams) (config RequesterConfig) {
	var err error

	defer func() {
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize compute config %s", err.Error()))
		}
	}()
	if params.JobNegotiationTimeout == 0 {
		params.JobNegotiationTimeout = DefaultRequesterConfig.JobNegotiationTimeout
	}
	if params.MinJobExecutionTimeout == 0 {
		params.MinJobExecutionTimeout = DefaultRequesterConfig.MinJobExecutionTimeout
	}
	if params.DefaultJobExecutionTimeout == 0 {
		params.DefaultJobExecutionTimeout = DefaultRequesterConfig.DefaultJobExecutionTimeout
	}
	if params.StateManagerBackgroundTaskInterval == 0 {
		params.StateManagerBackgroundTaskInterval = DefaultRequesterConfig.StateManagerBackgroundTaskInterval
	}
	if params.NodeRankRandomnessRange == 0 {
		params.NodeRankRandomnessRange = DefaultRequesterConfig.NodeRankRandomnessRange
	}
	if params.NodeInfoStoreTTL == 0 {
		params.NodeInfoStoreTTL = DefaultRequesterConfig.NodeInfoStoreTTL
	}
	if params.DiscoveredPeerStoreTTL == 0 {
		params.DiscoveredPeerStoreTTL = DefaultRequesterConfig.DiscoveredPeerStoreTTL
	}

	config = RequesterConfig{
		JobNegotiationTimeout:      params.JobNegotiationTimeout,
		MinJobExecutionTimeout:     params.MinJobExecutionTimeout,
		DefaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,

		StateManagerBackgroundTaskInterval: params.StateManagerBackgroundTaskInterval,

		NodeRankRandomnessRange: params.NodeRankRandomnessRange,
		NodeInfoStoreTTL:        params.NodeInfoStoreTTL,
		DiscoveredPeerStoreTTL:  params.DiscoveredPeerStoreTTL,
		SimulatorConfig:         params.SimulatorConfig,
	}

	return config
}
