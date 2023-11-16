package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type BaseEndpointParams struct {
	ID                string
	EvaluationBroker  EvaluationBroker
	Store             jobstore.Store
	EventEmitter      EventEmitter
	ComputeProxy      compute.Endpoint
	JobTransformer    transformer.JobTransformer
	ResultTransformer transformer.ResultTransformer
}

type BaseEndpoint struct {
	id                string
	evaluationBroker  EvaluationBroker
	store             jobstore.Store
	eventEmitter      EventEmitter
	computeProxy      compute.Endpoint
	jobTransformer    transformer.JobTransformer
	resultTransformer transformer.ResultTransformer
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	return &BaseEndpoint{
		id:                params.ID,
		evaluationBroker:  params.EvaluationBroker,
		store:             params.Store,
		eventEmitter:      params.EventEmitter,
		computeProxy:      params.ComputeProxy,
		jobTransformer:    params.JobTransformer,
		resultTransformer: params.ResultTransformer,
	}
}

// SubmitJob submits a job to the evaluation broker.
func (e *BaseEndpoint) SubmitJob(ctx context.Context, request *SubmitJobRequest) (*SubmitJobResponse, error) {
	job := request.Job
	job.Normalize()
	warnings := job.SanitizeSubmission()

	if err := e.jobTransformer.Transform(ctx, job); err != nil {
		return nil, err
	}

	if err := e.store.CreateJob(ctx, *job); err != nil {
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
	if err := e.store.CreateEvaluation(ctx, *eval); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to save evaluation for job %s", job.ID)
		return nil, err
	}

	if err := e.evaluationBroker.Enqueue(eval); err != nil {
		return nil, err
	}
	e.eventEmitter.EmitJobCreated(ctx, *job)
	return &SubmitJobResponse{
		JobID:        job.ID,
		EvaluationID: eval.ID,
		Warnings:     warnings,
	}, nil
}

func (e *BaseEndpoint) StopJob(ctx context.Context, request *StopJobRequest) (StopJobResponse, error) {
	job, err := e.store.GetJob(ctx, request.JobID)
	if err != nil {
		return StopJobResponse{}, err
	}
	switch job.State.StateType {
	case models.JobStateTypeStopped:
		// no need to stop a job that is already stopped
		return StopJobResponse{}, nil
	case models.JobStateTypeCompleted:
		return StopJobResponse{}, fmt.Errorf("cannot stop job in state %s", job.State.StateType)
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
		Comment:  request.Reason,
	})
	if err != nil {
		return StopJobResponse{}, err
	}

	// enqueue evaluation to allow the scheduler to stop existing executions
	// if the job is not terminal already, such as failed
	evalID := ""
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
			log.Ctx(ctx).Error().Err(err).Msgf("failed to save evaluation for stop job %s", request.JobID)
			return StopJobResponse{}, err
		}

		err = e.evaluationBroker.Enqueue(eval)
		if err != nil {
			return StopJobResponse{}, err
		}
		evalID = eval.ID
	}
	e.eventEmitter.EmitEventSilently(ctx, model.JobEvent{
		JobID:     request.JobID,
		EventName: model.JobEventCanceled,
		Status:    request.Reason,
		EventTime: time.Now(),
	})
	return StopJobResponse{
		EvaluationID: evalID,
	}, nil
}

func (e *BaseEndpoint) ReadLogs(ctx context.Context, request ReadLogsRequest) (ReadLogsResponse, error) {
	emptyResponse := ReadLogsResponse{}

	executions, err := e.store.GetExecutions(ctx, request.JobID)
	if err != nil {
		return emptyResponse, err
	}

	nodeID := ""
	for _, exec := range executions {
		if exec.ID == request.ExecutionID {
			nodeID = exec.NodeID
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

	response, err := e.computeProxy.ExecutionLogs(ctx, req)
	if err != nil {
		return emptyResponse, err
	}

	return ReadLogsResponse{
		Address:           response.Address,
		ExecutionComplete: response.ExecutionFinished,
	}, nil
}

// GetResults returns the results of a job
func (e *BaseEndpoint) GetResults(ctx context.Context, request *GetResultsRequest) (GetResultsResponse, error) {
	job, err := e.store.GetJob(ctx, request.JobID)
	if err != nil {
		return GetResultsResponse{}, err
	}

	if job.Type != models.JobTypeBatch && job.Type != models.JobTypeOps {
		return GetResultsResponse{}, fmt.Errorf("job type %s does not support results", job.Type)
	}

	executions, err := e.store.GetExecutions(ctx, job.ID)
	if err != nil {
		return GetResultsResponse{}, err
	}

	results := make([]*models.SpecConfig, 0)
	for _, execution := range executions {
		if execution.ComputeState.StateType == models.ExecutionStateCompleted {
			result := execution.PublishedResult.Copy()
			err = e.resultTransformer.Transform(ctx, result)
			if err != nil {
				return GetResultsResponse{}, err
			}
			results = append(results, result)
		}
	}

	return GetResultsResponse{
		Results: results,
	}, nil
}
