package requester

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/jobtransform"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type BaseEndpointParams struct {
	ID                         string
	PublicKey                  []byte
	EvaluationBroker           orchestrator.EvaluationBroker
	Store                      jobstore.Store
	EvaluationQueue            *EvaluationQueue
	EventEmitter               orchestrator.EventEmitter
	ComputeEndpoint            compute.Endpoint
	StorageProviders           storage.StorageProvider
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration
}

// BaseEndpoint base implementation of requester Endpoint
type BaseEndpoint struct {
	id               string
	evaluationBroker orchestrator.EvaluationBroker
	store            jobstore.Store
	evaluationQueue  *EvaluationQueue
	eventEmitter     orchestrator.EventEmitter
	computesvc       compute.Endpoint
	transforms       []jobtransform.Transformer
	postTransforms   []jobtransform.PostTransformer
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	transforms := []jobtransform.Transformer{
		jobtransform.NewTimeoutApplier(params.MinJobExecutionTimeout, params.DefaultJobExecutionTimeout),
		jobtransform.NewRequesterInfo(params.ID, params.PublicKey),
		jobtransform.RepoExistsOnIPFS(params.StorageProviders),
		jobtransform.NewPublisherMigrator(),
		jobtransform.NewEngineMigrator(),
		// jobtransform.DockerImageDigest(),
	}

	postTransforms := []jobtransform.PostTransformer{
		jobtransform.NewWasmStorageSpecConverter(),
		jobtransform.NewInlineStoragePinner(params.StorageProviders),
	}

	return &BaseEndpoint{
		id:               params.ID,
		evaluationBroker: params.EvaluationBroker,
		computesvc:       params.ComputeEndpoint,
		store:            params.Store,
		evaluationQueue:  params.EvaluationQueue,
		transforms:       transforms,
		postTransforms:   postTransforms,
		eventEmitter:     params.EventEmitter,
	}
}

func (e *BaseEndpoint) SubmitJob(ctx context.Context, data model.JobCreatePayload) (*model.Job, error) {
	jobUUID, err := uuid.NewRandom()
	if err != nil {
		return &model.Job{}, fmt.Errorf("error creating job id: %w", err)
	}
	jobID := jobUUID.String()

	// Creates a new root context to track a job's lifecycle for tracing. This
	// should be fine as only one node will call SubmitJob(...) - the other
	// nodes will hear about the job via events on the transport.
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester.BaseEndpoint.SubmitJob",
		// job lifecycle spans go in their own, dedicated trace
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, e.id),
			attribute.String(model.TracerAttributeNameJobID, jobID),
		),
	)
	defer span.End()

	// TODO: Should replace the span above, with the below, but I don't understand how/why we're tracing contexts in a variable.
	// Specifically tracking them all in ctrl.jobContexts
	// ctx, span := system.NewRootSpan(ctx, system.GetTracer(), "pkg/controller.SubmitJob")
	// defer span.End()

	now := time.Now().UTC()
	legacyJob := &model.Job{
		APIVersion: data.APIVersion,
		Metadata: model.Metadata{
			ID:        jobID,
			ClientID:  data.ClientID,
			CreatedAt: now,
		},
		Spec: *data.Spec,
	}

	for _, transform := range e.transforms {
		_, err = transform(ctx, legacyJob)
		if err != nil {
			return nil, err
		}
	}

	// convert to new job model
	job, err := legacy.FromLegacyJob(legacyJob)
	if err != nil {
		return nil, err
	}

	for _, transform := range e.postTransforms {
		_, err = transform(ctx, job)
		if err != nil {
			return nil, err
		}
	}

	err = e.store.CreateJob(ctx, *job)
	if err != nil {
		return nil, err
	}

	eval := &models.Evaluation{
		ID:          uuid.NewString(),
		JobID:       job.ID,
		TriggeredBy: models.EvalTriggerJobRegister,
		Type:        job.Type,
		Status:      models.EvalStatusPending,
		CreateTime:  job.CreateTime,
		ModifyTime:  job.CreateTime,
	}

	// TODO(ross): How can we create this evaluation in the same transaction that the CreateJob
	// call uses.
	err = e.store.CreateEvaluation(ctx, *eval)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to save evaluation for job %s", jobID)
		return nil, err
	}

	// err = e.evaluationBroker.Enqueue(eval)
	// if err != nil {
	// 	return nil, err
	// }

	e.eventEmitter.EmitJobCreated(ctx, *job)
	return legacyJob, nil
}

func (e *BaseEndpoint) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	job, err := e.store.GetJob(ctx, request.JobID)
	if err != nil {
		return CancelJobResult{}, err
	}
	switch job.State.StateType {
	case models.JobStateTypeStopped:
		// no need to cancel a job that is already stopped
		return CancelJobResult{}, nil
	case models.JobStateTypeCompleted:
		return CancelJobResult{}, fmt.Errorf("cannot cancel job in state %s", job.State.StateType)
	}

	// update the job state, except if the job is already completed
	// we allow marking a failed job as canceled
	err = e.store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
		JobID: request.JobID,
		Condition: jobstore.UpdateJobCondition{
			UnexpectedStates: []models.JobStateType{
				models.JobStateTypeCompleted,
			},
		},
		NewState: models.JobStateTypeStopped,
		Comment:  "job canceled by user",
	})
	if err != nil {
		return CancelJobResult{}, err
	}

	// enqueue evaluation to allow the scheduler to cancel existing executions
	// if the job is not terminal already, such as failed
	if !job.IsTerminal() {
		now := time.Now().UTC().UnixNano()
		eval := &models.Evaluation{
			ID:          uuid.NewString(),
			JobID:       request.JobID,
			TriggeredBy: models.EvalTriggerJobCancel,
			Type:        job.Type,
			Status:      models.EvalStatusPending,
			CreateTime:  now,
			ModifyTime:  now,
		}

		// TODO(ross): How can we create this evaluation in the same transaction that we update the jobstate
		err = e.store.CreateEvaluation(ctx, *eval)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to save evaluation for cancel job %s", request.JobID)
			return CancelJobResult{}, err
		}

		// err = e.evaluationBroker.Enqueue(eval)
		// if err != nil {
		// 	return CancelJobResult{}, err
		// }
	}
	e.eventEmitter.EmitEventSilently(ctx, model.JobEvent{
		JobID:     request.JobID,
		EventName: model.JobEventCanceled,
		Status:    request.Reason,
		EventTime: time.Now(),
	})
	return CancelJobResult{}, nil
}

func (e *BaseEndpoint) ReadLogs(ctx context.Context, request ReadLogsRequest) (ReadLogsResponse, error) {
	emptyResponse := ReadLogsResponse{}

	executions, err := e.store.GetExecutions(ctx, request.JobID)
	if err != nil {
		return emptyResponse, err
	}

	nodeID := ""
	for _, e := range executions {
		if e.ID == request.ExecutionID {
			nodeID = e.NodeID
			break
		}
	}

	if nodeID == "" {
		return emptyResponse, fmt.Errorf("unable to find execution %s in job %s", request.ExecutionID, request.JobID)
	}

	req := compute.ExecutionLogsRequest{
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: e.id,
			TargetPeerID: nodeID,
		},
		ExecutionID: request.ExecutionID,
		WithHistory: request.WithHistory,
		Follow:      request.Follow,
	}

	newCtx := context.Background()
	response, err := e.computesvc.ExecutionLogs(newCtx, req)
	if err != nil {
		return emptyResponse, err
	}

	return ReadLogsResponse{Address: response.Address, ExecutionComplete: response.ExecutionFinished}, nil
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
			ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted).WithMessage(response.Reason),
		},
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

	// err = e.evaluationBroker.Enqueue(eval)
	// if err != nil {
	// 	log.Ctx(ctx).Error().Err(err).Msgf("[%s] failed to enqueue evaluation for job %s", operation, jobID)
	// }
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
var _ compute.Callback = (*BaseEndpoint)(nil)
