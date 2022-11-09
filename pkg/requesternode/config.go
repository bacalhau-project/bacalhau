package requesternode

import (
	"time"
)

// DefaultStateTransitionTimeout default timeout value for each shard state, except running state
const DefaultStateTransitionTimeout = 1 * time.Minute

// DefaultJobExecutionTimeout default timeout value for running a job, if the user didn't define one in the spec
const DefaultJobExecutionTimeout = 5 * time.Minute

// DefaultMinJobExecutionTimeout requester node will replace any job execution timeout that is less than
// this value with DefaultJobExecutionTimeout.
const DefaultMinJobExecutionTimeout = 0 * time.Second

// DefaultStateManagerTaskInterval background task interval that periodically checks for expired states among other things.
const DefaultStateManagerTaskInterval = 30 * time.Second

type RequesterTimeoutConfig struct {
	// Timeout value for each shard state, except running state
	StateTransitionTimeout time.Duration

	// Timeout value for running a job, if the user didn't define one in the spec
	DefaultJobExecutionTimeout time.Duration

	// Requester node will replace any job execution timeout that is less than this
	// value with DefaultJobExecutionTimeout.
	MinJobExecutionTimeout time.Duration
}

func NewDefaultRequesterTimeoutConfig() RequesterTimeoutConfig {
	return RequesterTimeoutConfig{
		StateTransitionTimeout:     DefaultStateTransitionTimeout,
		DefaultJobExecutionTimeout: DefaultJobExecutionTimeout,
		MinJobExecutionTimeout:     DefaultMinJobExecutionTimeout,
	}
}

type RequesterNodeConfig struct {
	// configure the timeout for each shard state
	TimeoutConfig RequesterTimeoutConfig

	// background task interval that periodically checks for expired states among other things.
	StateManagerBackgroundTaskInterval time.Duration
}

func NewDefaultRequesterNodeConfig() RequesterNodeConfig {
	return RequesterNodeConfig{
		TimeoutConfig:                      NewDefaultRequesterTimeoutConfig(),
		StateManagerBackgroundTaskInterval: DefaultStateManagerTaskInterval,
	}
}

func populateDefaultConfigs(other RequesterNodeConfig) RequesterNodeConfig {
	config := other

	if config.TimeoutConfig.StateTransitionTimeout == 0 {
		config.TimeoutConfig.StateTransitionTimeout = DefaultStateTransitionTimeout
	}
	if config.TimeoutConfig.DefaultJobExecutionTimeout == 0 {
		config.TimeoutConfig.DefaultJobExecutionTimeout = DefaultJobExecutionTimeout
	}
	if config.StateManagerBackgroundTaskInterval == 0 {
		config.StateManagerBackgroundTaskInterval = DefaultStateManagerTaskInterval
	}

	return config
}
