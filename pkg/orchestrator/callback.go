package orchestrator

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

type CallbackParams struct {
	ID    string
	Store jobstore.Store
}

// Callback base implementation of requester Endpoint
type Callback struct {
	id    string
	store jobstore.Store
}

func NewCallback(params *CallbackParams) *Callback {
	return &Callback{
		id:    params.ID,
		store: params.Store,
	}
}

// /////////////////////////////
// Compute callback handlers //
// /////////////////////////////

// OnBidComplete implements compute.Callback
func (e *Callback) OnBidComplete(ctx context.Context, response legacy.BidResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node received bid response %+v", response)

	var executionEvents []models.Event

	updateRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: response.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				models.ExecutionStateAskForBid,
				models.ExecutionStateNew, // in case the compute node responded before the compute_forwarder updated the execution state
			},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted).
				WithMessage(response.Event.Message).
				WithDetails(response.Event.Details),
		},
	}

	if !response.Accepted {
		updateRequest.NewValues.ComputeState.StateType = models.ExecutionStateAskForBidRejected
		updateRequest.NewValues.DesiredState =
			models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("bid rejected")
		executionEvents = append(executionEvents, response.Event)
	}

	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnBidComplete] failed to begin transaction")
		return
	}

	defer txContext.Rollback() //nolint:errcheck

	if err = e.store.UpdateExecution(txContext, updateRequest); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnBidComplete] failed to update execution")
		return
	}

	if response.JobVersion == 0 {
		log.Ctx(ctx).Warn().Msgf(
			"[OnBidComplete] received bid response for job %s, but job version is 0 (Old Compute Node?)",
			response.JobID,
		)
	}
	if err = e.store.AddExecutionHistory(
		txContext,
		response.JobID,
		response.JobVersion,
		response.ExecutionID,
		executionEvents...,
	); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnBidComplete] failed to add execution history")
		return
	}

	// enqueue evaluation to allow the scheduler to either accept the bid, or find a new node
	e.enqueueEvaluation(txContext, response.JobID, "OnBidComplete")

	if err = txContext.Commit(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnBidComplete] failed to commit transaction")
		return
	}
}

func (e *Callback) OnRunComplete(ctx context.Context, result legacy.RunResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received RunComplete for execution: %s from %s",
		e.id, result.ExecutionID, result.SourcePeerID)

	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to begin transaction")
		return
	}

	defer txContext.Rollback() //nolint:errcheck

	job, err := e.store.GetJob(txContext, result.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to get job %s", result.JobID)
		return
	}

	// update execution state
	updateRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: result.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				// usual expected state
				models.ExecutionStateBidAccepted,
				// in case of approval is required, and the compute node responded before
				// the compute_forwarder updated the execution state
				models.ExecutionStateAskForBidAccepted,
				// in case of approval is not required, and the compute node responded before
				// the compute_forwarder updated the execution state
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
		updateRequest.NewValues.ComputeState =
			models.NewExecutionState(models.ExecutionStateFailed).WithMessage("execution completed unexpectedly")
		updateRequest.NewValues.DesiredState =
			models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("execution completed unexpectedly")
	}

	if err = e.store.UpdateExecution(txContext, updateRequest); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to update execution")
		return
	}

	if err = e.store.AddExecutionHistory(txContext, result.JobID, result.JobVersion, result.ExecutionID, ExecCompletedEvent()); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to add execution history")
		return
	}

	// enqueue evaluation to allow the scheduler to mark the job as completed if all executions are completed
	e.enqueueEvaluation(txContext, result.JobID, "OnRunComplete")

	if err = txContext.Commit(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnComputeFailure] failed to commit transaction")
		return
	}
}

func (e *Callback) OnComputeFailure(ctx context.Context, result legacy.ComputeError) {
	log.Ctx(ctx).Debug().Err(result).Msgf("Requester node %s received ComputeFailure for execution: %s from %s",
		e.id, result.ExecutionID, result.SourcePeerID)

	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to begin transaction")
		return
	}

	defer txContext.Rollback() //nolint:errcheck

	// update execution state
	if err = e.store.UpdateExecution(txContext, jobstore.UpdateExecutionRequest{
		ExecutionID: result.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			UnexpectedStates: []models.ExecutionStateType{
				models.ExecutionStateCompleted,
				models.ExecutionStateCancelled,
			},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateFailed).
				WithMessage(result.Error()).
				WithDetails(result.Event.Details),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("execution failed"),
		},
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnComputeFailure] failed to update execution")
		return
	}

	// enqueue evaluation to allow the scheduler find other nodes, or mark the job as failed
	e.enqueueEvaluation(txContext, result.JobID, "OnComputeFailure")

	if err = e.store.AddExecutionHistory(txContext, result.JobID, result.JobVersion, result.ExecutionID, result.Event); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnComputeFailure] failed to add execution history")
		return
	}

	if err = txContext.Commit(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnComputeFailure] failed to commit transaction")
		return
	}
}

// enqueueEvaluation enqueues an evaluation to allow the scheduler to either accept the bid, or find a new node
func (e *Callback) enqueueEvaluation(ctx context.Context, jobID, operation string) {
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
var _ compute.Callback = (*Callback)(nil)
