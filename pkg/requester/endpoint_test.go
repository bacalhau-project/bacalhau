package requester

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	noop_verifier "github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
	"github.com/stretchr/testify/require"
)

type mockBidStrategy bool

// ShouldBid implements bidstrategy.BidStrategy
func (m *mockBidStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.BidStrategyResponse{ShouldBid: bool(*m)}, nil
}

// ShouldBidBasedOnUsage implements bidstrategy.BidStrategy
func (m *mockBidStrategy) ShouldBidBasedOnUsage(ctx context.Context, request bidstrategy.BidStrategyRequest, resourceUsage model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.BidStrategyResponse{ShouldBid: bool(*m)}, nil
}

var _ bidstrategy.BidStrategy = (*mockBidStrategy)(nil)

func TestEndpointAppliesJobSelectionPolicy(t *testing.T) {
	type errRequire func(require.TestingT, error, ...interface{})

	cm := system.NewCleanupManager()
	verifier_mock, err := noop_verifier.NewNoopVerifier(context.Background(), cm)
	require.NoError(t, err)
	storage_mock := noop_storage.NewNoopStorage(noop_storage.StorageConfig{})
	require.NoError(t, err)

	runTest := func(shouldBid bool, check errRequire) {
		strategy := mockBidStrategy(shouldBid)
		endpoint := NewBaseEndpoint(&BaseEndpointParams{
			Scheduler:        &mockScheduler{},
			Selector:         &strategy,
			Verifiers:        model.NewNoopProvider[model.Verifier, verifier.Verifier](verifier_mock),
			StorageProviders: model.NewNoopProvider[model.StorageSourceType, storage.Storage](storage_mock),
		})

		job, err := endpoint.SubmitJob(context.Background(), model.JobCreatePayload{
			Spec: &model.Spec{
				Network: model.NetworkConfig{
					Type: model.NetworkFull,
				},
			},
		})

		require.NotNil(t, job)
		check(t, err)
	}

	t.Run("should not accept when strategy returns false", func(t *testing.T) { runTest(false, require.Error) })
	t.Run("should accept when strategy returns true", func(t *testing.T) { runTest(true, require.NoError) })
}
