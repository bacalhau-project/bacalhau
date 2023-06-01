package node

import (
	"fmt"
	"net/url"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
)

type RequesterConfigParams struct {
	// Timeout config
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration

	HousekeepingBackgroundTaskInterval time.Duration
	NodeRankRandomnessRange            int
	OverAskForBidsFactor               uint
	JobSelectionPolicy                 model.JobSelectionPolicy
	ExternalValidatorWebhook           *url.URL
	SimulatorConfig                    model.SimulatorConfigRequester

	// minimum version of compute nodes that the requester will accept and route jobs to
	MinBacalhauVersion model.BuildVersionInfo

	RetryStrategy requester.RetryStrategy
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
	NodeRankRandomnessRange  int
	OverAskForBidsFactor     uint
	JobSelectionPolicy       model.JobSelectionPolicy
	ExternalValidatorWebhook *url.URL
	SimulatorConfig          model.SimulatorConfigRequester

	// minimum version of compute nodes that the requester will accept and route jobs to
	MinBacalhauVersion model.BuildVersionInfo

	RetryStrategy requester.RetryStrategy
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
	if params.OverAskForBidsFactor == 0 {
		params.OverAskForBidsFactor = DefaultRequesterConfig.OverAskForBidsFactor
	}
	if params.MinBacalhauVersion == (model.BuildVersionInfo{}) {
		params.MinBacalhauVersion = DefaultRequesterConfig.MinBacalhauVersion
	}

	config = RequesterConfig{
		MinJobExecutionTimeout:             params.MinJobExecutionTimeout,
		DefaultJobExecutionTimeout:         params.DefaultJobExecutionTimeout,
		HousekeepingBackgroundTaskInterval: params.HousekeepingBackgroundTaskInterval,
		JobSelectionPolicy:                 params.JobSelectionPolicy,
		NodeRankRandomnessRange:            params.NodeRankRandomnessRange,
		OverAskForBidsFactor:               params.OverAskForBidsFactor,
		ExternalValidatorWebhook:           params.ExternalValidatorWebhook,
		SimulatorConfig:                    params.SimulatorConfig,
		MinBacalhauVersion:                 params.MinBacalhauVersion,
		RetryStrategy:                      params.RetryStrategy,
	}

	return config
}
