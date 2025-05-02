package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bacalhau-project/bacalhau/pkg/analytics"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
)

type BaseEndpointParams struct {
	ID                string
	Store             jobstore.Store
	LogstreamServer   logstream.Server
	JobTransformer    transformer.JobTransformer
	ResultTransformer transformer.ResultTransformer
}

type BaseEndpoint struct {
	id                string
	store             jobstore.Store
	logstreamServer   logstream.Server
	jobTransformer    transformer.JobTransformer
	resultTransformer transformer.ResultTransformer
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	return &BaseEndpoint{
		id:                params.ID,
		store:             params.Store,
		logstreamServer:   params.LogstreamServer,
		jobTransformer:    params.JobTransformer,
		resultTransformer: params.ResultTransformer,
	}
}

// SubmitJob submits a job to the evaluation broker.
func (e *BaseEndpoint) SubmitJob(ctx context.Context, request *SubmitJobRequest) (_ *SubmitJobResponse, err error) {
	job := request.Job
	job.Normalize()
	warnings := job.SanitizeSubmission()

	var jobID string
	defer func() {
		analytics.Emit(analytics.NewSubmitJobEvent(*job, jobID, err, warnings...))
	}()

	if request.ClientInstallationID != "" {
		job.Meta[models.MetaClientInstallationID] = request.ClientInstallationID
	}
	if request.ClientInstanceID != "" {
		job.Meta[models.MetaClientInstanceID] = request.ClientInstanceID
	}

	if err = e.jobTransformer.Transform(ctx, job); err != nil {
		return nil, err
	}

	// set jobId for telemetry purposes
	jobID = job.ID

	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer txContext.Rollback() //nolint:errcheck

	if err = e.store.CreateJob(txContext, *job); err != nil {
		return nil, err
	}
	if err = e.store.AddJobHistory(txContext, job.ID, JobSubmittedEvent()); err != nil {
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

	if err = e.store.CreateEvaluation(txContext, *eval); err != nil {
		return nil, err
	}

	if err = txContext.Commit(); err != nil {
		return nil, err
	}

	return &SubmitJobResponse{
		JobID:        job.ID,
		EvaluationID: eval.ID,
		Warnings:     warnings,
	}, nil
}

func (e *BaseEndpoint) StopJob(ctx context.Context, request *StopJobRequest) (StopJobResponse, error) {
	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return StopJobResponse{}, jobstore.NewJobStoreError(err.Error())
	}
	defer txContext.Rollback() //nolint:errcheck

	job, err := e.store.GetJob(txContext, request.JobID)
	if err != nil {
		return StopJobResponse{}, err
	}
	switch job.State.StateType {
	case models.JobStateTypeStopped:
		// no need to stop a job that is already stopped
		return StopJobResponse{}, nil
	case models.JobStateTypeCompleted:
		return StopJobResponse{}, bacerrors.Newf("cannot stop job in state %s", job.State.StateType)
	default:
		// continue
	}

	// update the job state, except if the job is already completed
	// we allow marking a failed job as canceled
	if err = e.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
		JobID: job.ID, // use the job ID from the store in case the request had a short ID
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
			JobID:       job.ID,
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

	return StopJobResponse{
		EvaluationID: evalID,
	}, nil
}

// 1. Find the compute node on which the execution was run (regardless of its version).
// 2. Ask it for logs.
// 3. If it fails to provide logs:
//   - For terminal executions that have un-truncated logs in RunOutput, return them.
//   - For non-terminal executions, return an error.
func (e *BaseEndpoint) ReadLogs(ctx context.Context, request ReadLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	// TODO: Handle the case when job is running, but there are no executions yet (e.g. they still sit in the queue).
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

		// TODO: A job can have a single cancelled execution that produced logs.
		// 		We want to be able to read those.
	}

	if execution == nil {
		return nil, fmt.Errorf("unable to find execution %s in job %s", request.ExecutionID, request.JobID)
	}

	req := messages.ExecutionLogsRequest{
		ExecutionID: execution.ID,
		NodeID:      execution.NodeID,
		Tail:        request.Tail,
		Follow:      request.Follow,
	}

	// TODO: If the target node is not reachable or fails to provide logs, we should either return an error
	// 		or fallback to logs from the execution run result (for terminal executions)
	return e.logstreamServer.GetLogStream(ctx, req)
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
