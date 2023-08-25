package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
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
	localExecStates, err := s.executionStore.GetLiveExecutions(ctx)
	if err != nil {
		return err
	}

	errs := new(multierror.Error)

	for idx := range localExecStates {
		localExecution := localExecStates[idx]

		switch localExecution.Execution.Job.Type {
		case models.JobTypeService, models.JobTypeDaemon:
			{
				// Service and System jobs are long running jobs and so we need to make sure it is running
				err = s.runExecution(ctx, localExecution)
				if err != nil {
					errs = multierror.Append(errs, err)
				}
			}
		case model.JobTypeBatch, models.JobTypeOps:
			{
				// Batch and Ops jobs should be failed as we don't know if they had any
				// side-effects (particularly for ops jobs).
				err = s.failExecution(ctx, localExecution)
				if err != nil {
					errs = multierror.Append(errs, err)
				}
			}
		}
	}

	return errs.Unwrap()
}

func (s *Startup) failExecution(ctx context.Context, execution store.LocalExecutionState) error {
	// Calling cancel with the execute buffer will update our local state, and
	// then when it calls the underlying baseexecutor, that will inform the requester
	// node. We really want to _Fail_ the execution, but that's currently only possible
	// as part of a call to .Run().
	err := s.execBuffer.Cancel(ctx, execution)
	if err != nil {
		return err
	}

	return err
}

func (s *Startup) runExecution(ctx context.Context, execution store.LocalExecutionState) error {
	// We want to ensure this execution is running.  If we just call .Run, there's a
	// chance we'll end up with two copies running if the previous version is alive
	// for whatever reason.  We'll call Cancel here so that we can be sure that it
	// is not running before we ask for it to be run.
	// TODO: Find a better way of either:
	// * Findout out whether the underlying process (e.g. docker) is still running, or
	// * Have .Run be idempotent for the execution id without relying on the store.
	_ = s.execBuffer.Cancel(ctx, execution)

	log.Ctx(ctx).Info().Msgf("Re-running execution %s after restart", execution.Execution.ID)
	return s.execBuffer.Run(ctx, execution)
}
