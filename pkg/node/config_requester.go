package node

import (
	"fmt"
	"net/url"
	"time"

	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
)

type RequesterConfigParams struct {
	JobDefaults transformer.JobDefaults

	HousekeepingBackgroundTaskInterval time.Duration
	HousekeepingTimeoutBuffer          time.Duration
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

	// Scheduler config
	NodeOverSubscriptionFactor float64

	// Should the orchestrator attempt to translate jobs?
	TranslationEnabled bool

	S3PreSignedURLDisabled   bool
	S3PreSignedURLExpiration time.Duration

	JobStore jobstore.Store

	DefaultPublisher string

	// When new nodes join the cluster, what state do they have? By default, APPROVED, and
	// for tests, APPROVED. We will provide an option to set this to PENDING for production
	// or for when operators are ready to control node approval.
	DefaultApprovalState models.NodeMembershipState

	ControlPlaneSettings types.RequesterControlPlaneConfig
}

type RequesterConfig struct {
	RequesterConfigParams
}

func NewRequesterConfigWithDefaults() (RequesterConfig, error) {
	return NewRequesterConfigWith(getRequesterConfigParams())
}

//nolint:gosimple
func NewRequesterConfigWith(params RequesterConfigParams) (RequesterConfig, error) {
	defaults := getRequesterConfigParams()
	if err := mergo.Merge(&params, defaults); err != nil {
		return RequesterConfig{}, fmt.Errorf("creating requester config: %w", err)
	}

	// TODO: move away from how we define approval states as they don't have clear
	//  zero value and don't play nicely with merge
	if params.DefaultApprovalState.IsUndefined() {
		params.DefaultApprovalState = defaults.DefaultApprovalState
	}

	log.Debug().Msgf("Requester config: %+v", params)
	return RequesterConfig{
		RequesterConfigParams: params,
	}, nil
}
