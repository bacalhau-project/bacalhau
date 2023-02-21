package compute

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/util/generic"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
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
	cancellers      generic.SyncMap[store.Execution, context.CancelFunc]
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

// Run the execution of a shard after it has been accepted, and propose a result to the requester to be verified.
func (e *BaseExecutor) Run(ctx context.Context, execution store.Execution) (err error) {
	ctx = log.Ctx(ctx).With().
		Str("Shard", execution.Shard.ID()).
		Str("ExecutionID", execution.ID).
		Logger().WithContext(ctx)

	defer func() {
		if err != nil {
			e.handleFailure(ctx, execution, err, "Running")
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	e.cancellers.Put(execution, cancel)
	defer func() {
		if cancel, found := e.cancellers.Get(execution); found {
			e.cancellers.Delete(execution)
			cancel()
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

	jobVerifier, err := e.verifiers.Get(ctx, execution.Shard.Job.Spec.Verifier)
	if err != nil {
		return
	}

	resultFolder, err := jobVerifier.GetShardResultPath(ctx, execution.Shard)
	if err != nil {
		return
	}

	jobExecutor, err := e.executors.Get(ctx, execution.Shard.Job.Spec.Engine)
	if err != nil {
		return
	}

	var runCommandResult *model.RunCommandResult

	if !e.simulatorConfig.IsBadActor {
		runCommandResult, err = jobExecutor.RunShard(ctx, execution.Shard, resultFolder)
		if err != nil {
			jobsFailed.Add(ctx, 1)
		} else {
			jobsCompleted.Add(ctx, 1)
		}

		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to run shard")
			return
		}
	}

	shardProposal, err := jobVerifier.GetShardProposal(ctx, execution.Shard, resultFolder)
	if err != nil {
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
		ResultProposal:   shardProposal,
		RunCommandResult: runCommandResult,
	})
	return err
}

// Publish the result of a shard execution after it has been verified.
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
	jobVerifier, err := e.verifiers.Get(ctx, execution.Shard.Job.Spec.Verifier)
	if err != nil {
		return
	}
	resultFolder, err := jobVerifier.GetShardResultPath(ctx, execution.Shard)
	if err != nil {
		return
	}
	jobPublisher, err := e.publishers.Get(ctx, execution.Shard.Job.Spec.Publisher)
	if err != nil {
		return
	}
	publishedResult, err := jobPublisher.PublishShardResult(ctx, execution.Shard, e.ID, resultFolder)
	if err != nil {
		return
	}

	err = e.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStatePublishing,
		NewState:      store.ExecutionStateCompleted,
	})
	if err != nil {
		return
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

// Cancel the execution of a running shard.
func (e *BaseExecutor) Cancel(ctx context.Context, execution store.Execution) (err error) {
	defer func() {
		if err != nil {
			e.handleFailure(ctx, execution, err, "Canceling")
		}
	}()

	log.Ctx(ctx).Debug().Str("execution", execution.ID).Msg("Canceling execution")
	if cancel, found := e.cancellers.Get(execution); found {
		e.cancellers.Delete(execution)
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
		log.Ctx(ctx).Error().Err(updateError).Msgf("Failed to update execution state to failed: %s", execution)
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
