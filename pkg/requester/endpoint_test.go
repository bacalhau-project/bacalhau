//go:build unit || !integration

package requester

import (
	"context"
	"net/url"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	noop_verifier "github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
	"github.com/stretchr/testify/require"
)

type mockBidStrategy struct {
	response bidstrategy.BidStrategyResponse
}

// ShouldBid implements bidstrategy.BidStrategy
func (m *mockBidStrategy) ShouldBid(context.Context, bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return m.response, nil
}

// ShouldBidBasedOnUsage implements bidstrategy.BidStrategy
func (m *mockBidStrategy) ShouldBidBasedOnUsage(context.Context, bidstrategy.BidStrategyRequest, model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
	return m.response, nil
}

var _ bidstrategy.BidStrategy = (*mockBidStrategy)(nil)

func getTestEndpoint(t *testing.T, strategy bidstrategy.BidStrategy) (Endpoint, jobstore.Store) {
	cm := system.NewCleanupManager()
	t.Cleanup(func() { cm.Cleanup(context.Background()) })

	verifier_mock, err := noop_verifier.NewNoopVerifier(context.Background(), cm)
	require.NoError(t, err)
	storage_mock := noop_storage.NewNoopStorage()
	require.NoError(t, err)
	store := inmemory.NewInMemoryJobStore()
	scheduler := &mockScheduler{
		handleStartJob: func(ctx context.Context, sjr StartJobRequest) error {
			store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID:    sjr.Job.Metadata.ID,
				NewState: model.JobStateInProgress,
			})
			return nil
		},
	}

	emitter := NewEventEmitter(EventEmitterParams{
		EventConsumer: eventhandler.JobEventHandlerFunc(func(ctx context.Context, event model.JobEvent) error {
			return nil
		}),
	})
	endpoint := NewBaseEndpoint(&BaseEndpointParams{
		Queue:              NewQueue(store, scheduler, emitter),
		Selector:           strategy,
		Store:              store,
		Verifiers:          model.NewNoopProvider[model.Verifier, verifier.Verifier](verifier_mock),
		StorageProviders:   model.NewNoopProvider[model.StorageSourceType, storage.Storage](storage_mock),
		GetBiddingCallback: func() *url.URL { return nil },
	})

	return endpoint, store
}

func TestEndpointAppliesJobSelectionPolicy(t *testing.T) {
	runTest := func(t *testing.T, shouldBid, shouldWait bool, expected model.JobStateType) {
		strategy := mockBidStrategy{
			response: bidstrategy.BidStrategyResponse{
				ShouldBid:  shouldBid,
				ShouldWait: shouldWait,
			},
		}

		endpoint, store := getTestEndpoint(t, &strategy)
		job, err := endpoint.SubmitJob(context.Background(), model.JobCreatePayload{
			Spec: &model.Spec{
				Network: model.NetworkConfig{
					Type: model.NetworkFull,
				},
			},
		})
		require.NotNil(t, job)
		require.NoError(t, err)

		state, err := store.GetJobState(context.Background(), job.Metadata.ID)
		require.NoError(t, err)
		require.Equal(t, expected, state.State)
	}

	t.Run("cancels when strategy returns false", func(t *testing.T) {
		runTest(t, false, false, model.JobStateCancelled)
	})

	t.Run("starts when strategy returns true", func(t *testing.T) {
		runTest(t, true, false, model.JobStateInProgress)
	})

	t.Run("queues when strategy says to wait", func(t *testing.T) {
		runTest(t, false, true, model.JobStateQueued)
		runTest(t, true, true, model.JobStateQueued)
	})
}

func TestEndpointAcceptsApprovals(t *testing.T) {
	runTest := func(t *testing.T, shouldBid bool, expected model.JobStateType) {
		strategy := mockBidStrategy{
			response: bidstrategy.BidStrategyResponse{ShouldWait: true},
		}
		endpoint, store := getTestEndpoint(t, &strategy)

		job, err := endpoint.SubmitJob(context.Background(), model.JobCreatePayload{
			Spec: &model.Spec{},
		})
		require.NotNil(t, job)
		require.NoError(t, err)

		state, err := store.GetJobState(context.Background(), job.Metadata.ID)
		require.NoError(t, err)
		require.Equal(t, model.JobStateQueued, state.State)

		err = endpoint.ApproveJob(context.Background(), bidstrategy.ModerateJobRequest{
			ClientID: "",
			JobID:    job.Metadata.ID,
			Response: bidstrategy.BidStrategyResponse{ShouldBid: shouldBid},
		})

		state, err = store.GetJobState(context.Background(), job.Metadata.ID)
		require.NoError(t, err)
		require.Equal(t, expected, state.State)
	}

	t.Run("starts job when approving", func(t *testing.T) {
		runTest(t, true, model.JobStateInProgress)
	})

	t.Run("cancels job when rejecting", func(t *testing.T) {
		runTest(t, false, model.JobStateCancelled)
	})

	t.Run("rejects unknown client", func(t *testing.T) {
		t.Setenv("BACALHAU_JOB_APPROVER", "hello")
		runTest(t, false, model.JobStateQueued)
		runTest(t, true, model.JobStateQueued)
	})
}
