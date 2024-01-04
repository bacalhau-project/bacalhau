package node

import (
	"fmt"
	"net/url"
	"time"

	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
)

type RequesterConfigParams struct {
	JobDefaults transformer.JobDefaults

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

	// Should the orchestrator attempt to translate jobs?
	TranslationEnabled bool

	S3PreSignedURLDisabled   bool
	S3PreSignedURLExpiration time.Duration
}

type RequesterConfig struct {
	RequesterConfigParams
}

func NewRequesterConfigWithDefaults() (RequesterConfig, error) {
	return NewRequesterConfigWith(getRequesterConfigParams())
}

//nolint:gosimple
func NewRequesterConfigWith(params RequesterConfigParams) (RequesterConfig, error) {
	if err := mergo.Merge(&params, getRequesterConfigParams()); err != nil {
		return RequesterConfig{}, fmt.Errorf("creating requester config: %w", err)
	}

	log.Debug().Msgf("Requester config: %+v", params)
	return RequesterConfig{
		RequesterConfigParams: params,
	}, nil
}
