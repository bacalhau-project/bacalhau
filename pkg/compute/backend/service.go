package backend

import (
	"context"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type BaseServiceParams struct {
	ID         string
	Callback   Callback
	Store      store.ExecutionStore
	Executors  executor.ExecutorProvider
	Verifiers  verifier.VerifierProvider
	Publishers publisher.PublisherProvider
}

// BaseService is the base implementation for backend service.
// All operations are executed asynchronously, and a callback is used to notify the caller of the result.
type BaseService struct {
	ID         string
	callback   Callback
	store      store.ExecutionStore
	executors  executor.ExecutorProvider
	verifiers  verifier.VerifierProvider
	publishers publisher.PublisherProvider
}

func NewBaseService(params BaseServiceParams) *BaseService {
	return &BaseService{
		ID:         params.ID,
		callback:   params.Callback,
		store:      params.Store,
		executors:  params.Executors,
		verifiers:  params.Verifiers,
		publishers: params.Publishers,
	}
}

// Run the execution of a shard after it has been accepted, and propose a result to the requester to be verified.
func (s BaseService) Run(ctx context.Context, execution store.Execution) (err error) {
	defer func() {
		if err != nil {
			s.callback.OnRunFailure(ctx, execution.ID, err)
		}
	}()

	log.Ctx(ctx).Debug().Msgf("Running execution %s", execution.ID)
	err = s.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateBidAccepted,
		NewState:      store.ExecutionStateRunning,
	})
	if err != nil {
		return
	}

	jobVerifier, err := s.verifiers.GetVerifier(ctx, execution.Shard.Job.Spec.Verifier)
	if err != nil {
		return
	}

	resultFolder, err := jobVerifier.GetShardResultPath(ctx, execution.Shard)
	if err != nil {
		return
	}

	jobExecutor, err := s.executors.GetExecutor(ctx, execution.Shard.Job.Spec.Engine)
	if err != nil {
		return
	}
	runCommandResult, err := jobExecutor.RunShard(ctx, execution.Shard, resultFolder)
	if err != nil {
		jobsFailed.With(prometheus.Labels{
			"node_id":     s.ID,
			"shard_index": strconv.Itoa(execution.Shard.Index),
			"client_id":   execution.Shard.Job.ClientID,
		}).Inc()
	} else {
		jobsCompleted.With(prometheus.Labels{
			"node_id":     s.ID,
			"shard_index": strconv.Itoa(execution.Shard.Index),
			"client_id":   execution.Shard.Job.ClientID,
		}).Inc()
	}

	if err != nil {
		return
	}

	shardProposal, err := jobVerifier.GetShardProposal(ctx, execution.Shard, resultFolder)
	if err != nil {
		return
	}

	s.callback.OnRunSuccess(ctx, execution.ID, RunResult{
		ResultProposal:   shardProposal,
		RunCommandResult: runCommandResult,
	})
	return err
}

// Publish the result of a shard execution after it has been verified.
func (s BaseService) Publish(ctx context.Context, execution store.Execution) (err error) {
	defer func() {
		if err != nil {
			s.callback.OnPublishFailure(ctx, execution.ID, err)
		}
	}()
	log.Ctx(ctx).Debug().Msgf("Publishing execution %s", execution.ID)
	err = s.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   execution.ID,
		ExpectedState: store.ExecutionStateResultAccepted,
		NewState:      store.ExecutionStatePublishing,
	})
	if err != nil {
		return
	}
	jobVerifier, err := s.verifiers.GetVerifier(ctx, execution.Shard.Job.Spec.Verifier)
	if err != nil {
		return
	}
	resultFolder, err := jobVerifier.GetShardResultPath(ctx, execution.Shard)
	if err != nil {
		return
	}
	jobPublisher, err := s.publishers.GetPublisher(ctx, execution.Shard.Job.Spec.Publisher)
	if err != nil {
		return
	}
	publishedResult, err := jobPublisher.PublishShardResult(ctx, execution.Shard, s.ID, resultFolder)
	if err != nil {
		return
	}
	s.callback.OnPublishSuccess(ctx, execution.ID, PublishResult{
		PublishResult: publishedResult,
	})
	return err
}

// Cancel the execution of a running shard.
func (s BaseService) Cancel(ctx context.Context, execution store.Execution) (err error) {
	defer func() {
		if err != nil {
			s.callback.OnCancelFailure(ctx, execution.ID, err)
		}
	}()

	log.Ctx(ctx).Debug().Msgf("Canceling execution %s", execution.ID)
	// check that we have the executor to cancel this job
	jobExecutor, err := s.executors.GetExecutor(ctx, execution.Shard.Job.Spec.Engine)
	if err != nil {
		return
	}
	err = jobExecutor.CancelShard(ctx, execution.Shard)
	if err != nil {
		return err
	}
	s.callback.OnCancelSuccess(ctx, execution.ID, CancelResult{})
	return
}

// compile-time interface check
var _ Service = (*BaseService)(nil)
