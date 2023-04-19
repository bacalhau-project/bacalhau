package compute

import (
	"context"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

type BaseExecutorParams struct {
	ID              string
	Callback        Callback
	Store           store.ExecutionStore
	Executors       executor.ExecutorProvider
	Verifiers       verifier.VerifierProvider
	Publishers      publisher.PublisherProvider
	SimulatorConfig model.SimulatorConfigCompute
}

// BaseExecutor is the base implementation for backend service.
// All operations are executed asynchronously, and a callback is used to notify the caller of the result.
type BaseExecutor struct {
	ID              string
	callback        Callback
	store           store.ExecutionStore
	cancellers      generic.SyncMap[string, context.CancelFunc]
	executors       executor.ExecutorProvider
	verifiers       verifier.VerifierProvider
	publishers      publisher.PublisherProvider
	simulatorConfig model.SimulatorConfigCompute
}

func NewBaseExecutor(params BaseExecutorParams) *BaseExecutor {
	return &BaseExecutor{
		ID:              params.ID,
		callback:        params.Callback,
		store:           params.Store,
		executors:       params.Executors,
		verifiers:       params.Verifiers,
		publishers:      params.Publishers,
		simulatorConfig: params.SimulatorConfig,
	}
}

// Run the execution after it has been accepted, and propose a result to the requester to be verified.
func (e *BaseExecutor) Run(ctx context.Context, execution store.Execution) (err error) {
	ctx = log.Ctx(ctx).With().
		Str("job", execution.Job.ID()).
		Str("execution", execution.ID).
		Logger().WithContext(ctx)

	ctx, cancel := context.WithCancel(ctx)
	e.cancellers.Put(execution.ID, cancel)
	defer func() {
		if cancel, found := e.cancellers.Get(execution.ID); found {
			e.cancellers.Delete(execution.ID)
			cancel()
		}
	}()

	defer func() {
		if err != nil {
			e.handleFailure(ctx, execution, err, "Running")
		}
	}()

	log.Ctx(ctx).Debug().Msg("Running execution")
	err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateBidAccepted,
		NewState:      store.ExecutionStateRunning,
	})
	if err != nil {
		return
	}

	jobVerifier, err := e.verifiers.Get(ctx, execution.Job.Spec.Verifier)
	if err != nil {
		err = fmt.Errorf("failed to get verifier %s: %w", execution.Job.Spec.Verifier, err)
		return
	}

	resultFolder, err := jobVerifier.GetResultPath(ctx, execution.ID, execution.Job)
	if err != nil {
		err = fmt.Errorf("failed to get result path: %w", err)
		return
	}

	jobExecutor, err := e.executors.Get(ctx, execution.Job.Spec.Engine)
	if err != nil {
		err = fmt.Errorf("failed to get executor %s: %w", execution.Job.Spec.Engine, err)
		return
	}

	var runCommandResult *model.RunCommandResult

	if !e.simulatorConfig.IsBadActor {
		runCommandResult, err = jobExecutor.Run(ctx, execution.ID, execution.Job, resultFolder)
		if err != nil {
			jobsFailed.Add(ctx, 1)
		} else {
			jobsCompleted.Add(ctx, 1)
		}

		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to run execution")
			return
		}
	}

	proposal, err := jobVerifier.GetProposal(ctx, execution.Job, resultFolder)
	if err != nil {
		err = fmt.Errorf("failed to get proposal: %w", err)
		return
	}

	err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateRunning,
		NewState:      store.ExecutionStateWaitingVerification,
	})
	if err != nil {
		return
	}

	e.callback.OnRunComplete(ctx, RunResult{
		ExecutionMetadata: NewExecutionMetadata(execution),
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: e.ID,
			TargetPeerID: execution.RequesterNodeID,
		},
		ResultProposal:   proposal,
		RunCommandResult: runCommandResult,
	})
	return err
}

// Publish the result of an execution after it has been verified.
func (e *BaseExecutor) Publish(ctx context.Context, execution store.Execution) (err error) {
	defer func() {
		if err != nil {
			e.handleFailure(ctx, execution, err, "Publishing")
		}
	}()
	log.Ctx(ctx).Debug().Msgf("Publishing execution %s", execution.ID)
	err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateResultAccepted,
		NewState:      store.ExecutionStatePublishing,
	})
	if err != nil {
		return
	}
	jobVerifier, err := e.verifiers.Get(ctx, execution.Job.Spec.Verifier)
	if err != nil {
		err = fmt.Errorf("failed to get verifier %s: %w", execution.Job.Spec.Verifier, err)
		return
	}
	resultFolder, err := jobVerifier.GetResultPath(ctx, execution.ID, execution.Job)
	if err != nil {
		err = fmt.Errorf("failed to get result path: %w", err)
		return
	}
	jobPublisher, err := e.publishers.Get(ctx, execution.Job.Spec.PublisherSpec.Type)
	if err != nil {
		err = fmt.Errorf("failed to get publisher %s: %w", execution.Job.Spec.PublisherSpec.Type, err)
		return
	}
	publishedResult, err := jobPublisher.PublishResult(ctx, execution.ID, execution.Job, resultFolder)
	if err != nil {
		err = fmt.Errorf("failed to publish result: %w", err)
		return
	}

	log.Ctx(ctx).Debug().
		Str("execution", execution.ID).
		Str("cid", publishedResult.CID).
		Msg("Execution published")

	err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStatePublishing,
		NewState:      store.ExecutionStateCompleted,
	})
	if err != nil {
		return
	}

	log.Ctx(ctx).Debug().Msgf("Cleaning up result folder for %s: %s", execution.ID, resultFolder)
	err = os.RemoveAll(resultFolder)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to remove results folder at %s", resultFolder)
	}

	e.callback.OnPublishComplete(ctx, PublishResult{
		ExecutionMetadata: NewExecutionMetadata(execution),
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: e.ID,
			TargetPeerID: execution.RequesterNodeID,
		},
		PublishResult: publishedResult,
	})
	return err
}

// Cancel the execution.
func (e *BaseExecutor) Cancel(ctx context.Context, execution store.Execution) (err error) {
	defer func() {
		if err != nil {
			e.handleFailure(ctx, execution, err, "Canceling")
		}
	}()

	log.Ctx(ctx).Debug().Str("Execution", execution.ID).Msg("Canceling execution")
	if cancel, found := e.cancellers.Get(execution.ID); found {
		e.cancellers.Delete(execution.ID)
		cancel()
	}

	e.callback.OnCancelComplete(ctx, CancelResult{
		ExecutionMetadata: NewExecutionMetadata(execution),
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: e.ID,
			TargetPeerID: execution.RequesterNodeID,
		},
	})
	return err
}

func (e *BaseExecutor) handleFailure(ctx context.Context, execution store.Execution, err error, operation string) {
	log.Ctx(ctx).Error().Err(err).Msgf("%s execution %s failed", operation, execution.ID)
	updateError := e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: execution.ID,
		NewState:    store.ExecutionStateFailed,
		Comment:     err.Error(),
	})

	if updateError != nil {
		log.Ctx(ctx).Error().Err(updateError).Msgf("Failed to update execution (%s) state to failed: %s", execution.ID, updateError)
	} else {
		e.callback.OnComputeFailure(ctx, ComputeError{
			ExecutionMetadata: NewExecutionMetadata(execution),
			RoutingMetadata: RoutingMetadata{
				SourcePeerID: e.ID,
				TargetPeerID: execution.RequesterNodeID,
			},
			Err: err.Error(),
		})
	}
}

// compile-time interface check
var _ Executor = (*BaseExecutor)(nil)
