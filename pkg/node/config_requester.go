package node

import (
	"fmt"
	"net/url"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"
)

type RequesterConfigParams struct {
	// Timeout config
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration

	HousekeepingBackgroundTaskInterval time.Duration
	NodeRankRandomnessRange            int
	OverAskForBidsFactor               uint
	JobSelectionPolicy                 JobSelectionPolicy
	ExternalValidatorWebhook           *url.URL
	FailureInjectionConfig             model.FailureInjectionRequesterConfig

	// minimum version of compute nodes that the requester will accept and route jobs to
	MinBacalhauVersion models.BuildVersionInfo

	RetryStrategy orchestrator.RetryStrategy

	// evaluation broker config
	EvalBrokerVisibilityTimeout    time.Duration
	EvalBrokerInitialRetryDelay    time.Duration
	EvalBrokerSubsequentRetryDelay time.Duration
	EvalBrokerMaxRetryCount        int

	// worker config
	WorkerCount                  int
	WorkerEvalDequeueTimeout     time.Duration
	WorkerEvalDequeueBaseBackoff time.Duration
	WorkerEvalDequeueMaxBackoff  time.Duration
}

type RequesterConfig struct {
	RequesterConfigParams
}

func NewRequesterConfigWithDefaults() RequesterConfig {
	return NewRequesterConfigWith(getRequesterConfigParams())
}

//nolint:gosimple
func NewRequesterConfigWith(params RequesterConfigParams) (config RequesterConfig) {
	var err error

	defer func() {
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize compute config %s", err.Error()))
		}
	}()

	err = mergo.Merge(&params, getRequesterConfigParams())
	if err != nil {
		return
	}

	log.Debug().Msgf("Requester config: %+v", params)
	return RequesterConfig{
		RequesterConfigParams: params,
	}
}
