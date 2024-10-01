//go:build unit || !integration

package serve

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestGetComputeConfigTest(t *testing.T) {
	ctx := context.Background()

	cfg := types.Default
	localPublisherPort := 9999
	localPublisherAddress := "1.1.1.1"
	localPublisherDirectory := "/publisher"
	cfg.Publishers.Types.Local.Port = localPublisherPort
	cfg.Publishers.Types.Local.Address = localPublisherAddress
	cfg.Publishers.Types.Local.Directory = localPublisherDirectory

	createExecutionStore := true

	computeNodeConfig, err := GetComputeConfig(ctx, cfg, createExecutionStore)
	require.NoError(t, err)

	// values with no corresponding config field
	assert.False(t, computeNodeConfig.IgnorePhysicalResourceLimits)
	assert.Equal(t, 3*time.Minute, computeNodeConfig.JobNegotiationTimeout)
	assert.Equal(t, 500*time.Millisecond, computeNodeConfig.MinJobExecutionTimeout)
	assert.Equal(t, models.NoTimeout, computeNodeConfig.MaxJobExecutionTimeout)
	assert.Equal(t, models.NoTimeout, computeNodeConfig.DefaultJobExecutionTimeout)
	assert.Equal(t, semantic.Anywhere, computeNodeConfig.JobSelectionPolicy.Locality)
	assert.Equal(t, 10*time.Second, computeNodeConfig.LogRunningExecutionsInterval)
	assert.Equal(t, 0, computeNodeConfig.LogStreamBufferSize)
	assert.Equal(t, false, computeNodeConfig.FailureInjectionConfig.IsBadActor)
	assert.Nil(t, computeNodeConfig.BidSemanticStrategy)
	assert.Nil(t, computeNodeConfig.BidSemanticStrategy)

	// we can't check the path from this interface against the config, so assert its been initalized
	assert.NotNil(t, computeNodeConfig.ExecutionStore)

	// fields with a direct mapping to the config
	assert.Equal(t, cfg.JobAdmissionControl.RejectStatelessJobs, computeNodeConfig.JobSelectionPolicy.RejectStatelessJobs)
	assert.Equal(t, cfg.JobAdmissionControl.AcceptNetworkedJobs, computeNodeConfig.JobSelectionPolicy.AcceptNetworkedJobs)
	assert.Equal(t, cfg.JobAdmissionControl.ProbeHTTP, computeNodeConfig.JobSelectionPolicy.ProbeHTTP)
	assert.Equal(t, cfg.JobAdmissionControl.ProbeExec, computeNodeConfig.JobSelectionPolicy.ProbeHTTP)

	assert.Equal(t, cfg.Publishers.Types.Local.Port, computeNodeConfig.LocalPublisher.Port)
	assert.Equal(t, cfg.Publishers.Types.Local.Address, computeNodeConfig.LocalPublisher.Address)
	assert.Equal(t, cfg.Publishers.Types.Local.Directory, computeNodeConfig.LocalPublisher.Directory)

	assert.EqualValues(t, cfg.Compute.Heartbeat.Interval, computeNodeConfig.ControlPlaneSettings.HeartbeatFrequency)
	assert.EqualValues(t, cfg.Compute.Heartbeat.InfoUpdateInterval, computeNodeConfig.ControlPlaneSettings.InfoUpdateFrequency)
	assert.EqualValues(t, cfg.Compute.Heartbeat.ResourceUpdateInterval, computeNodeConfig.ControlPlaneSettings.ResourceUpdateFrequency)
}
