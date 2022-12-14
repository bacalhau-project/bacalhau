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
	SimulatorConfig         model.SimulatorConfigRequester
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

	config = RequesterConfig{
		JobNegotiationTimeout:      params.JobNegotiationTimeout,
		MinJobExecutionTimeout:     params.MinJobExecutionTimeout,
		DefaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,

		StateManagerBackgroundTaskInterval: params.StateManagerBackgroundTaskInterval,

		NodeRankRandomnessRange: params.NodeRankRandomnessRange,
		SimulatorConfig:         params.SimulatorConfig,
	}

	return config
}
