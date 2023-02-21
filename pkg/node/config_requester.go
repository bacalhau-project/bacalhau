package node

import (
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type RequesterConfigParams struct {
	// Timeout config
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration

	HousekeepingBackgroundTaskInterval time.Duration
	NodeRankRandomnessRange            int
	SimulatorConfig                    model.SimulatorConfigRequester
}

type RequesterConfig struct {
	// MinJobExecutionTimeout requester will replace any job execution timeout that is less than this
	// value with DefaultJobExecutionTimeout.
	MinJobExecutionTimeout time.Duration
	// DefaultJobExecutionTimeout default value for running, verifying and publishing job results,
	// if the user didn't define one in the spec
	DefaultJobExecutionTimeout time.Duration

	// HousekeepingBackgroundTaskInterval background task interval that periodically checks for expired states
	HousekeepingBackgroundTaskInterval time.Duration
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
	if params.MinJobExecutionTimeout == 0 {
		params.MinJobExecutionTimeout = DefaultRequesterConfig.MinJobExecutionTimeout
	}
	if params.DefaultJobExecutionTimeout == 0 {
		params.DefaultJobExecutionTimeout = DefaultRequesterConfig.DefaultJobExecutionTimeout
	}
	if params.HousekeepingBackgroundTaskInterval == 0 {
		params.HousekeepingBackgroundTaskInterval = DefaultRequesterConfig.HousekeepingBackgroundTaskInterval
	}
	if params.NodeRankRandomnessRange == 0 {
		params.NodeRankRandomnessRange = DefaultRequesterConfig.NodeRankRandomnessRange
	}

	config = RequesterConfig{
		MinJobExecutionTimeout:     params.MinJobExecutionTimeout,
		DefaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,

		HousekeepingBackgroundTaskInterval: params.HousekeepingBackgroundTaskInterval,

		NodeRankRandomnessRange: params.NodeRankRandomnessRange,
		SimulatorConfig:         params.SimulatorConfig,
	}

	return config
}
