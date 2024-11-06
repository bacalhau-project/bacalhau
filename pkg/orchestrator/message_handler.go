package orchestrator

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

// MessageHandler base implementation of requester Endpoint
type MessageHandler struct {
	store jobstore.Store
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(store jobstore.Store) *MessageHandler {
	return &MessageHandler{
		store: store,
	}
}

func (m *MessageHandler) ShouldProcess(ctx context.Context, message *ncl.Message) bool {
	return message.Metadata.Get(ncl.KeyMessageType) == messages.BidResultMessageType ||
		message.Metadata.Get(ncl.KeyMessageType) == messages.RunResultMessageType ||
		message.Metadata.Get(ncl.KeyMessageType) == messages.ComputeErrorMessageType
}

// HandleMessage handles incoming messages
// TODO: handle messages arriving out of order gracefully
func (m *MessageHandler) HandleMessage(ctx context.Context, message *ncl.Message) error {
	switch message.Metadata.Get(ncl.KeyMessageType) {
	case messages.BidResultMessageType:
		return m.OnBidComplete(ctx, message)
	case messages.RunResultMessageType:
		return m.OnRunComplete(ctx, message)
	case messages.ComputeErrorMessageType:
		return m.OnComputeFailure(ctx, message)
	default:
		return nil
	}
}

// OnBidComplete handles the completion of a bid request
func (m *MessageHandler) OnBidComplete(ctx context.Context, message *ncl.Message) error {
	result, ok := message.Payload.(*messages.BidResult)
	if !ok {
		return ncl.NewErrUnexpectedPayloadType("BidResult", reflect.TypeOf(message.Payload).String())
	}

	updateRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: result.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedDesiredStates: []models.ExecutionDesiredStateType{
				models.ExecutionDesiredStatePending, models.ExecutionDesiredStateRunning,
			},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted).WithMessage(result.Message()),
		},
		Events: result.Events,
	}

	if !result.Accepted {
		updateRequest.NewValues.ComputeState.StateType = models.ExecutionStateAskForBidRejected
		updateRequest.NewValues.DesiredState =
			models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("bid rejected")
	}

	txContext, err := m.store.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = txContext.Rollback()
	}()

	if err = m.store.UpdateExecution(txContext, updateRequest); err != nil {
		return err
	}

	// enqueue evaluation to allow the scheduler to either accept the bid, or find a new node
	err = m.enqueueEvaluation(txContext, result.JobID, result.JobType)
	if err != nil {
		return err
	}

	return txContext.Commit()
}

func (m *MessageHandler) OnRunComplete(ctx context.Context, message *ncl.Message) error {
	result, ok := message.Payload.(*messages.RunResult)
	if !ok {
		return ncl.NewErrUnexpectedPayloadType("RunResult", reflect.TypeOf(message.Payload).String())
	}

	txContext, err := m.store.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = txContext.Rollback()
	}()

	job, err := m.store.GetJob(txContext, result.JobID)
	if err != nil {
		return err
	}

	// update execution state
	updateRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: result.ExecutionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedDesiredStates: []models.ExecutionDesiredStateType{
				models.ExecutionDesiredStateRunning,
			},
		},
		NewValues: models.Execution{
			PublishedResult: result.PublishResult,
			RunOutput:       result.RunCommandResult,
			ComputeState:    models.NewExecutionState(models.ExecutionStateCompleted),
			DesiredState:    models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("execution completed"),
		},
		Events: result.Events,
	}

	if job.IsLongRunning() {
		log.Ctx(ctx).Error().Msgf(
			"job %s is long running, but received a RunComplete. Marking the execution as failed instead", result.JobID)
		updateRequest.NewValues.ComputeState =
			models.NewExecutionState(models.ExecutionStateFailed).WithMessage("execution completed unexpectedly")
		updateRequest.NewValues.DesiredState =
			models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped).WithMessage("execution completed unexpectedly")
	}

	if err = m.store.UpdateExecution(txContext, updateRequest); err != nil {
		return err
	}

	// enqueue evaluation to allow the scheduler to mark the job as completed if all executions are completed
	if err = m.enqueueEvaluation(txContext, result.JobID, result.JobType); err != nil {
		return err
	}

	return txContext.Commit()
}

func (m *MessageHandler) OnComputeFailure(ctx context.Context, message *ncl.Message) error {
	result, ok := message.Payload.(*messages.ComputeError)
	if !ok {
		return ncl.NewErrUnexpectedPayloadType("ComputeError", reflect.TypeOf(message.Payload).String())
	}

	txContext, err := m.store.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = txContext.Rollback()
	}()

	// update execution state
	if err = m.store.UpdateExecution(txContext, jobstore.UpdateExecutionRequest{
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
		Events: result.Events,
	}); err != nil {
		return err
	}

	// enqueue evaluation to allow the scheduler find other nodes, or mark the job as failed
	if err = m.enqueueEvaluation(txContext, result.JobID, result.JobType); err != nil {
		return err
	}

	return txContext.Commit()
}

// enqueueEvaluation enqueues an evaluation to allow the scheduler to either accept the bid, or find a new node
func (m *MessageHandler) enqueueEvaluation(ctx context.Context, jobID, jobType string) error {
	now := time.Now().UTC().UnixNano()
	eval := &models.Evaluation{
		ID:          uuid.NewString(),
		JobID:       jobID,
		TriggeredBy: models.EvalTriggerExecUpdate,
		Type:        jobType,
		Status:      models.EvalStatusPending,
		CreateTime:  now,
		ModifyTime:  now,
	}

	err := m.store.CreateEvaluation(ctx, *eval)
	if err != nil {
		return fmt.Errorf("failed to create/save evaluation for job %s: %w", jobID, err)
	}
	return nil
}
