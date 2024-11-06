package compute

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type Startup struct {
	executionStore store.ExecutionStore
	execBuffer     Executor
}

func NewStartup(execStore store.ExecutionStore, execBuffer Executor) *Startup {
	return &Startup{
		executionStore: execStore,
		execBuffer:     execBuffer,
	}
}

// Execute is used by the compute node to perform startup tasks
// that should happen before the node takes part in the rest of the
// network. This might be executions setup, or cleaning previous
// inputs etc.
func (s *Startup) Execute(ctx context.Context) error {
	log.Ctx(ctx).Debug().Msg("Performing startup tasks")

	err := s.ensureLiveJobs(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *Startup) ensureLiveJobs(ctx context.Context) error {
	log.Ctx(ctx).Debug().Msg("Startup: Checking live executions")

	// Get a list of the currently live executions and we can check their
	// status - relying on the changes to the execution state to update
	// the index.
	executions, err := s.executionStore.GetLiveExecutions(ctx)
	if err != nil {
		return err
	}

	var errs error

	for idx := range executions {
		execution := executions[idx]

		switch execution.Job.Type {
		case models.JobTypeService, models.JobTypeDaemon:
			{
				// Service and System jobs are long running jobs and so we need to make sure it is running
				err = s.runExecution(ctx, execution)
				if err != nil {
					errs = errors.Join(errs, err)
				}
			}
		case models.JobTypeBatch, models.JobTypeOps:
			{
				// Batch and Ops jobs should be failed as we don't know if they had any
				// side-effects (particularly for ops jobs).
				err = s.failExecution(ctx, execution)
				if err != nil {
					errs = errors.Join(errs, err)
				}
			}
		}
	}

	return errs
}

func (s *Startup) failExecution(ctx context.Context, execution *models.Execution) error {
	log.Ctx(ctx).Info().Msgf("Failing execution %s after restart", execution.ID)
	return s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateFailed).
				WithMessage("Failed due to node restart"),
		},
		Events: []*models.Event{ExecFailedDueToNodeRestartEvent()},
	})
}

func (s *Startup) runExecution(ctx context.Context, execution *models.Execution) error {
	// We want to ensure this 'live' execution is running and rather than go through
	// multiple steps trying to determine whether the executor or underlying process
	// is still running, we will just call Run() and expect it to do the correct thing.
	log.Ctx(ctx).Info().Msgf("Re-running execution %s after restart", execution.ID)
	return s.execBuffer.Run(ctx, execution)
}
