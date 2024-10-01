//go:build unit || !integration

package serve

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestGetRequesterConfig(t *testing.T) {
	cfg := types.Default
	createJobStore := true
	s3PreSignedExpiration := types.Duration(time.Second)
	cfg.Publishers.Types.S3.PreSignedURLExpiration = s3PreSignedExpiration
	requesterNodeConfig, err := GetRequesterConfig(cfg, createJobStore)
	require.NoError(t, err)

	assert.Equal(t, cfg.JobDefaults, requesterNodeConfig.JobDefaults)

	assert.EqualValues(t, cfg.Orchestrator.Scheduler.HousekeepingInterval, requesterNodeConfig.HousekeepingBackgroundTaskInterval)
	assert.EqualValues(t, cfg.Orchestrator.Scheduler.HousekeepingTimeout, requesterNodeConfig.HousekeepingTimeoutBuffer)

	assert.Equal(t, 5, requesterNodeConfig.NodeRankRandomnessRange)

	assert.EqualValues(t, 3, requesterNodeConfig.OverAskForBidsFactor)

	assert.Equal(t, semantic.Anywhere, requesterNodeConfig.JobSelectionPolicy.Locality)
	assert.Equal(t, cfg.JobAdmissionControl.RejectStatelessJobs, requesterNodeConfig.JobSelectionPolicy.RejectStatelessJobs)
	assert.Equal(t, cfg.JobAdmissionControl.AcceptNetworkedJobs, requesterNodeConfig.JobSelectionPolicy.AcceptNetworkedJobs)
	assert.Equal(t, cfg.JobAdmissionControl.ProbeHTTP, requesterNodeConfig.JobSelectionPolicy.ProbeHTTP)
	assert.Equal(t, cfg.JobAdmissionControl.ProbeExec, requesterNodeConfig.JobSelectionPolicy.ProbeExec)

	assert.Nil(t, requesterNodeConfig.ExternalValidatorWebhook)

	assert.False(t, requesterNodeConfig.FailureInjectionConfig.IsBadActor)

	assert.Equal(t, models.BuildVersionInfo{
		Major: "1", Minor: "0", GitVersion: "v1.0.4",
	}, requesterNodeConfig.MinBacalhauVersion)

	assert.Nil(t, requesterNodeConfig.RetryStrategy)

	assert.EqualValues(t, cfg.Orchestrator.EvaluationBroker.VisibilityTimeout, requesterNodeConfig.EvalBrokerVisibilityTimeout)
	assert.EqualValues(t, 100*time.Millisecond, requesterNodeConfig.EvalBrokerInitialRetryDelay)
	assert.EqualValues(t, 100*time.Millisecond, requesterNodeConfig.EvalBrokerSubsequentRetryDelay)
	assert.EqualValues(t, cfg.Orchestrator.EvaluationBroker.MaxRetryCount, requesterNodeConfig.EvalBrokerMaxRetryCount)

	assert.Equal(t, cfg.Orchestrator.Scheduler.WorkerCount, requesterNodeConfig.WorkerCount)
	assert.Equal(t, 200*time.Millisecond, requesterNodeConfig.WorkerEvalDequeueTimeout)
	assert.Equal(t, 20*time.Millisecond, requesterNodeConfig.WorkerEvalDequeueBaseBackoff)
	assert.Equal(t, 200*time.Millisecond, requesterNodeConfig.WorkerEvalDequeueMaxBackoff)

	assert.EqualValues(t, 0, requesterNodeConfig.SchedulerQueueBackoff)
	assert.EqualValues(t, 1.5, requesterNodeConfig.NodeOverSubscriptionFactor)

	assert.Equal(t, cfg.FeatureFlags.ExecTranslation, requesterNodeConfig.TranslationEnabled)

	assert.Equal(t, cfg.Publishers.Types.S3.PreSignedURLDisabled, requesterNodeConfig.S3PreSignedURLDisabled)
	assert.EqualValues(t, cfg.Publishers.Types.S3.PreSignedURLExpiration, requesterNodeConfig.S3PreSignedURLExpiration)

	assert.NotNil(t, requesterNodeConfig.JobStore)

	assert.Zero(t, requesterNodeConfig.NodeInfoStoreTTL)

	if cfg.Orchestrator.NodeManager.ManualApproval {
		assert.Equal(t, models.NodeMembership.PENDING, requesterNodeConfig.DefaultApprovalState)
	} else {
		assert.Equal(t, models.NodeMembership.APPROVED, requesterNodeConfig.DefaultApprovalState)
	}

	assert.EqualValues(t, cfg.Orchestrator.NodeManager.DisconnectTimeout, requesterNodeConfig.ControlPlaneSettings.NodeDisconnectedAfter)
	assert.EqualValues(t, 30*time.Second, requesterNodeConfig.ControlPlaneSettings.HeartbeatCheckFrequency)
	assert.EqualValues(t, "", requesterNodeConfig.ControlPlaneSettings.HeartbeatTopic)
}
