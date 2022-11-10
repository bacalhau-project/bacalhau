package computenode

import (
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
)

// DefaultJobNegotiationTimeout default timeout value to hold a bid for a job
const DefaultJobNegotiationTimeout = 3 * time.Minute

// DefaultMinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
// lower timeout requirements will not be bid on.
const DefaultMinJobExecutionTimeout = 500 * time.Millisecond

// DefaultMaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
// higher timeout requirements will not be bid on.
const DefaultMaxJobExecutionTimeout = 60 * time.Minute

// DefaultStateManagerTaskInterval background task interval that periodically checks for expired states among other things.
const DefaultStateManagerTaskInterval = 30 * time.Second

type ComputeTimeoutConfig struct {
	// Timeout value for holding a bid before it is accepted or rejected.
	JobNegotiationTimeout time.Duration

	// Minimum timeout value for running a job. Jobs with lower timeout requirements will not be bid on.
	MinJobExecutionTimeout time.Duration

	// Maximum timeout value for running a job. Jobs with higher timeout requirements will not be bid on.
	MaxJobExecutionTimeout time.Duration
}

func NewDefaultComputeTimeoutConfig() ComputeTimeoutConfig {
	return ComputeTimeoutConfig{
		JobNegotiationTimeout:  DefaultJobNegotiationTimeout,
		MinJobExecutionTimeout: DefaultMinJobExecutionTimeout,
		MaxJobExecutionTimeout: DefaultMaxJobExecutionTimeout,
	}
}

type ComputeNodeConfig struct {
	// this contains things like data locality and per
	// job resource limits
	JobSelectionPolicy JobSelectionPolicy

	// configure the resource capacity we are allowing for
	// this compute node
	CapacityManagerConfig capacitymanager.Config

	// configure the timeout for each shard state
	TimeoutConfig ComputeTimeoutConfig

	// background task interval that periodically checks for expired states among other things.
	StateManagerBackgroundTaskInterval time.Duration
}

func NewDefaultComputeNodeConfig() ComputeNodeConfig {
	return ComputeNodeConfig{
		JobSelectionPolicy:                 NewDefaultJobSelectionPolicy(),
		TimeoutConfig:                      NewDefaultComputeTimeoutConfig(),
		StateManagerBackgroundTaskInterval: DefaultStateManagerTaskInterval,
	}
}

func populateDefaultConfigs(other ComputeNodeConfig) ComputeNodeConfig {
	config := other

	if config.TimeoutConfig.JobNegotiationTimeout == 0 {
		config.TimeoutConfig.JobNegotiationTimeout = DefaultJobNegotiationTimeout
	}
	if config.StateManagerBackgroundTaskInterval == 0 {
		config.StateManagerBackgroundTaskInterval = DefaultStateManagerTaskInterval
	}

	return config
}
