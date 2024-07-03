package requester

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type BaseEndpointParams struct {
	ID           string
	Store        jobstore.Store
	EventEmitter orchestrator.EventEmitter
}

// BaseEndpoint base implementation of requester Endpoint
type BaseEndpoint struct {
	id           string
	store        jobstore.Store
	eventEmitter orchestrator.EventEmitter
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	return &BaseEndpoint{
		id:           params.ID,
		store:        params.Store,
		eventEmitter: params.EventEmitter,
	}
}

// /////////////////////////////
// Compute callback handlers //
// /////////////////////////////

// OnBidComplete implements compute.Callback
func (e *BaseEndpoint) OnBidComplete(ctx context.Context, response compute.BidResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node received bid response %+v", response)

	updateRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: response.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				models.ExecutionStateAskForBid,
				models.ExecutionStateNew, // in case the compute node responded before the compute_forwarder updated the execution state
			},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted).WithMessage(response.Event.Message),
		},
		Event: response.Event,
	}

	if !response.Accepted {
		updateRequest.NewValues.ComputeState.StateType = models.ExecutionStateAskForBidRejected
		updateRequest.NewValues.DesiredState =
			models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("bid rejected")
	}
	err := e.store.UpdateExecution(ctx, updateRequest)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnBidComplete] failed to update execution")
		return
	}

	// enqueue evaluation to allow the scheduler to either accept the bid, or find a new node
	e.enqueueEvaluation(ctx, response.JobID, "OnBidComplete")

	if response.Accepted {
		e.eventEmitter.EmitBidReceived(ctx, response)
	}
}

func (e *BaseEndpoint) OnRunComplete(ctx context.Context, result compute.RunResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received RunComplete for execution: %s from %s",
		e.id, result.ExecutionID, result.SourcePeerID)
	e.eventEmitter.EmitRunComplete(ctx, result)

	job, err := e.store.GetJob(ctx, result.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to get job %s", result.JobID)
		return
	}

	// update execution state
	updateExecutionRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: result.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				// usual expected state
				models.ExecutionStateBidAccepted,
				// in case of approval is required, and the compute node responded before the compute_forwarder updated the execution state
				models.ExecutionStateAskForBidAccepted,
				// in case of approval is not required, and the compute node responded before the compute_forwarder updated the execution state
				models.ExecutionStateNew,
			},
		},
		NewValues: models.Execution{
			PublishedResult: result.PublishResult,
			RunOutput:       result.RunCommandResult,
			ComputeState:    models.NewExecutionState(models.ExecutionStateCompleted),
			DesiredState:    models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("execution completed"),
		},
	}

	if job.IsLongRunning() {
		log.Ctx(ctx).Error().Msgf(
			"[OnRunComplete] job %s is long running, but received a RunComplete. Marking the execution as failed instead", result.JobID)
		updateExecutionRequest.NewValues.ComputeState =
			models.NewExecutionState(models.ExecutionStateFailed).WithMessage("execution completed unexpectedly")
		updateExecutionRequest.NewValues.DesiredState =
			models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("execution completed unexpectedly")
	}

	err = e.store.UpdateExecution(ctx, updateExecutionRequest)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to update execution")
		return
	}

	// enqueue evaluation to allow the scheduler to mark the job as completed if all executions are completed
	e.enqueueEvaluation(ctx, result.JobID, "OnRunComplete")
}

func (e *BaseEndpoint) OnCancelComplete(ctx context.Context, result compute.CancelResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received CancelComplete for execution: %s from %s",
		e.id, result.ExecutionID, result.SourcePeerID)
}

func (e *BaseEndpoint) OnComputeFailure(ctx context.Context, result compute.ComputeError) {
	log.Ctx(ctx).Debug().Err(result).Msgf("Requester node %s received ComputeFailure for execution: %s from %s",
		e.id, result.ExecutionID, result.SourcePeerID)

	// update execution state
	err := e.store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: result.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			UnexpectedStates: []models.ExecutionStateType{
				models.ExecutionStateCompleted,
				models.ExecutionStateCancelled,
			},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateFailed).WithMessage(result.Error()),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("execution failed"),
		},
		Event: result.Event,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnComputeFailure] failed to update execution")
		return
	}

	// enqueue evaluation to allow the scheduler find other nodes, or mark the job as failed
	e.enqueueEvaluation(ctx, result.JobID, "OnComputeFailure")
	e.eventEmitter.EmitComputeFailure(ctx, result.ExecutionID, result)
}

// enqueueEvaluation enqueues an evaluation to allow the scheduler to either accept the bid, or find a new node
// TODO: solve edge case where execution is updated, but evaluation is not enqueued
func (e *BaseEndpoint) enqueueEvaluation(ctx context.Context, jobID, operation string) {
	job, err := e.store.GetJob(ctx, jobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[%s] failed to get job %s while enqueueing evaluation", operation, jobID)
		return
	}
	now := time.Now().UTC().UnixNano()
	eval := &models.Evaluation{
		ID:          uuid.NewString(),
		JobID:       jobID,
		TriggeredBy: models.EvalTriggerExecUpdate,
		Type:        job.Type,
		Status:      models.EvalStatusPending,
		CreateTime:  now,
		ModifyTime:  now,
	}

	err = e.store.CreateEvaluation(ctx, *eval)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[%s] failed to create/save evaluation for job %s", operation, jobID)
		return
	}
}

// Compile-time interface check:
var _ compute.Callback = (*BaseEndpoint)(nil)
