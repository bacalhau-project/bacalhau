package watchers

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ExecutionCanceller struct {
	jobStore jobstore.Store
}

// NewExecutionCanceller creates a new NCL protocol dispatcher
func NewExecutionCanceller(jobStore jobstore.Store) *ExecutionCanceller {
	return &ExecutionCanceller{
		jobStore: jobStore,
	}
}

// HandleEvent implements watcher.EventHandler for NCL protocol
func (d *ExecutionCanceller) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(executionCancellerErrComponent)
	}

	// Skip if there's no state change
	if !upsert.HasStateChange() {
		return nil
	}

	transitions := newExecutionTransitions(upsert)
	if transitions.shouldCancel() {
		// TODO: should not update the execution's observed state here. Should listen for compute events instead.
		// Mark execution as cancelled
		if err := d.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
			ExecutionID: upsert.Current.ID,
			NewValues: models.Execution{
				ComputeState: models.State[models.ExecutionStateType]{
					StateType: models.ExecutionStateCancelled,
				},
			},
		}); err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Failed to mark execution %s as cancelled", upsert.Current.ID)
		}
	}

	return nil
}
