package orchestrator

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/pkg/analytics"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/translation"
)

type BaseEndpointParams struct {
	ID                string
	Store             jobstore.Store
	EventEmitter      EventEmitter
	ComputeProxy      compute.Endpoint
	JobTransformer    transformer.JobTransformer
	TaskTranslator    translation.TranslatorProvider
	ResultTransformer transformer.ResultTransformer
}

type BaseEndpoint struct {
	id                string
	store             jobstore.Store
	eventEmitter      EventEmitter
	computeProxy      compute.Endpoint
	jobTransformer    transformer.JobTransformer
	taskTranslator    translation.TranslatorProvider
	resultTransformer transformer.ResultTransformer
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	return &BaseEndpoint{
		id:                params.ID,
		store:             params.Store,
		eventEmitter:      params.EventEmitter,
		computeProxy:      params.ComputeProxy,
		jobTransformer:    params.JobTransformer,
		taskTranslator:    params.TaskTranslator,
		resultTransformer: params.ResultTransformer,
	}
}

// SubmitJob submits a job to the evaluation broker.
func (e *BaseEndpoint) SubmitJob(ctx context.Context, request *SubmitJobRequest) (_ *SubmitJobResponse, err error) {
	job := request.Job
	job.Normalize()
	warnings := job.SanitizeSubmission()
	submitEvent := analytics.NewSubmitJobEvent(*job, warnings...)
	defer func() {
		analytics.EmitEvent(ctx, analytics.NewEvent(analytics.SubmitJobEventType, submitEvent))
	}()

	if request.ClientInstallationID != "" {
		job.Meta[models.MetaClientInstallationID] = request.ClientInstallationID
	}
	if request.ClientInstanceID != "" {
		job.Meta[models.MetaClientInstanceID] = request.ClientInstanceID
	}

	if err := e.jobTransformer.Transform(ctx, job); err != nil {
		submitEvent.Error = err.Error()
		return nil, err
	}
	submitEvent.JobID = job.ID

	var translationEvent models.Event

	// We will only perform task translation in the orchestrator if we were provided with a provider
	// that can give translators to perform the translation.
	if e.taskTranslator != nil {
		// Before we create an evaluation for the job, we want to check that none of the job's tasks
		// need translating from a custom job type to a known job type (docker, wasm). If they do,
		// then we will perform the translation and create the evaluation for the new job instead.
		translatedJob, err := translation.Translate(ctx, e.taskTranslator, job)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to translate job type: %s", job.Task().Engine.Type))
		}

		// If we have translated the job (i.e. at least one task was translated) then we will record the original
		// job that was used to create the translated job. This will allow us to track the provenance of the job
		// when using `describe` and will ensure only the original job is returned when using `list`.
		if translatedJob != nil {
			if b, err := yaml.Marshal(translatedJob); err != nil {
				return nil, errors.Wrap(err, "failure converting job to JSON")
			} else {
				translatedJob.Meta[models.MetaDerivedFrom] = base64.StdEncoding.EncodeToString(b)
				translationEvent = JobTranslatedEvent(job, translatedJob)
			}

			job = translatedJob
		}
	}

	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = txContext.Rollback()
		}
	}()

	if err = e.store.CreateJob(txContext, *job); err != nil {
		return nil, err
	}
	if err = e.store.AddJobHistory(txContext, job.ID, JobSubmittedEvent()); err != nil {
		return nil, err
	}
	if translationEvent.Message != "" {
		if err = e.store.AddJobHistory(txContext, job.ID, translationEvent); err != nil {
			return nil, err
		}
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

	if err = e.store.CreateEvaluation(txContext, *eval); err != nil {
		return nil, err
	}

	if err = txContext.Commit(); err != nil {
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
		return StopJobResponse{}, models.NewBaseError("cannot stop job in state %s", job.State.StateType)
	default:
		// continue
	}

	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return StopJobResponse{}, jobstore.NewJobStoreError(err.Error())
	}

	defer func() {
		if err != nil {
			_ = txContext.Rollback()
		}
	}()

	// update the job state, except if the job is already completed
	// we allow marking a failed job as canceled
	if err = e.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
		JobID: request.JobID,
		Condition: jobstore.UpdateJobCondition{
			UnexpectedStates: []models.JobStateType{
				models.JobStateTypeCompleted,
			},
		},
		NewState: models.JobStateTypeStopped,
		Message:  request.Reason,
	}); err != nil {
		return StopJobResponse{}, err
	}

	if err = e.store.AddJobHistory(txContext, job.ID, JobStoppedEvent(request.Reason)); err != nil {
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

		if err = e.store.CreateEvaluation(txContext, *eval); err != nil {
			return StopJobResponse{}, err
		}
		evalID = eval.ID
	}

	if err = txContext.Commit(); err != nil {
		return StopJobResponse{}, err
	}

	e.eventEmitter.EmitEventSilently(ctx, models.JobEvent{
		JobID:     request.JobID,
		EventName: models.JobEventCanceled,
		Status:    request.Reason,
		EventTime: time.Now(),
	})
	return StopJobResponse{
		EvaluationID: evalID,
	}, nil
}

func (e *BaseEndpoint) ReadLogs(ctx context.Context, request ReadLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	executions, err := e.store.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: request.JobID,
	})
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return nil, fmt.Errorf("no executions found for job %s", request.JobID)
	}

	// TODO: support multiplexing logs from multiple executions. Might need a watermark to order the logs
	var execution *models.Execution
	var latestModifyTime int64 // zero time initially

	for i, exec := range executions {
		// If a specific execution ID is requested, select it directly
		if exec.ID == request.ExecutionID {
			execution = &executions[i]
			break
		}

		// If no specific execution is requested, track the latest non-discarded execution
		if request.ExecutionID == "" && !exec.IsDiscarded() && exec.ModifyTime > latestModifyTime {
			latestModifyTime = exec.ModifyTime
			execution = &executions[i]
		}
	}

	if execution == nil {
		return nil, fmt.Errorf("unable to find execution %s in job %s", request.ExecutionID, request.JobID)
	}

	if execution.IsTerminalState() {
		streamer := logstream.NewCompletedStreamer(logstream.CompletedStreamerParams{
			Execution: execution,
		})
		return streamer.Stream(ctx), nil
	}
	req := compute.ExecutionLogsRequest{
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: e.id,
			TargetPeerID: execution.NodeID,
		},
		ExecutionID: execution.ID,
		Tail:        request.Tail,
		Follow:      request.Follow,
	}

	return e.computeProxy.ExecutionLogs(ctx, req)
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

	executions, err := e.store.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: job.ID,
	})
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

			// Only add valid results
			if result.Type != "" {
				results = append(results, result)
			}
		}
	}

	return GetResultsResponse{
		Results: results,
	}, nil
}
