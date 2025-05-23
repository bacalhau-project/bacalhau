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
// Return the Job Version as well in the response
func (e *BaseEndpoint) SubmitJob(ctx context.Context, request *SubmitJobRequest) (_ *SubmitJobResponse, err error) {
	job := request.Job
	job.Normalize()

	// TODO: Implement warnings for the name syntax if it is not DNS compliant
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

	isUpdate := false
	evalTriggeredBy := models.EvalTriggerJobRegister
	existingJob, existingErr := e.store.GetJobByName(ctx, job.Name, job.Namespace)
	if existingErr == nil {
		// This is an update, the job name already exist
		job.ID = existingJob.ID
		isUpdate = true
		evalTriggeredBy = models.EvalTriggerJobRerun

		// Do a diff between jobs, and see if there is a difference
		if !request.Force {
			jobDiff := existingJob.CompareWith(job)
			if jobDiff == "" {
				return nil, bacerrors.Newf(
					"no changes detected for new job spec. Job Name: '%s', Job Id: '%s'",
					job.Name,
					job.ID,
				).WithHint("Use the --force flag to override this warning")
			}
		}
	}

	// set jobId for telemetry purposes
	jobID = job.ID

	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer txContext.Rollback() //nolint:errcheck

	// Create or update the job based on whether it's a new job or an update
	if isUpdate {
		if err = e.store.UpdateJob(txContext, *job); err != nil {
			return nil, err
		}

		if err = e.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
			JobID:    job.ID, // use the job ID from the store in case the request had a short ID
			NewState: models.JobStateTypePending,
			Message:  "",
		}); err != nil {
			return nil, err
		}

		// Add job history for the update, and bump the version number for this event.
		if err = e.store.AddJobHistory(txContext, job.ID, existingJob.Version+1, JobUpdatedEvent()); err != nil {
			return nil, err
		}
	} else {
		if err = e.store.CreateJob(txContext, *job); err != nil {
			return nil, err
		}

		if err = e.store.AddJobHistory(txContext, job.ID, 1, JobSubmittedEvent()); err != nil {
			return nil, err
		}
	}

	eval := &models.Evaluation{
		ID:          uuid.NewString(),
		JobID:       job.ID,
		TriggeredBy: evalTriggeredBy,
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

func (e *BaseEndpoint) DiffJob(ctx context.Context, request *DiffJobRequest) (_ *DiffJobResponse, err error) {
	job := request.Job
	job.Normalize()

	// TODO: Implement warnings for the name syntax if it is not DNS compliant
	warnings := job.SanitizeSubmission()

	if err = e.jobTransformer.Transform(ctx, job); err != nil {
		return nil, err
	}

	existingJob, existingErr := e.store.GetJobByName(ctx, job.Name, job.Namespace)
	if existingErr != nil {
		return nil, existingErr
	}

	job.ID = existingJob.ID
	jobDiff := existingJob.CompareWith(job)

	return &DiffJobResponse{
		Diff:     jobDiff,
		Warnings: warnings,
	}, nil
}

func (e *BaseEndpoint) StopJob(ctx context.Context, request *StopJobRequest) (StopJobResponse, error) {
	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return StopJobResponse{}, jobstore.NewJobStoreError(err.Error())
	}
	defer txContext.Rollback() //nolint:errcheck

	job, err := e.store.GetJobByIDOrName(txContext, request.JobID, request.Namespace)
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
		JobID: job.ID, // use the job ID from the store in case the request had a short ID, or it had a JobName
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

	if err = e.store.AddJobHistory(txContext, job.ID, job.Version, JobStoppedEvent(request.Reason)); err != nil {
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

func (e *BaseEndpoint) RerunJob(ctx context.Context, request *RerunJobRequest) (*RerunJobResponse, error) {
	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return &RerunJobResponse{}, jobstore.NewJobStoreError(err.Error())
	}
	job, err := e.store.GetJobByIDOrName(txContext, request.JobIDOrName, request.Namespace)
	if err != nil {
		return nil, jobstore.NewJobStoreError(err.Error())
	}
	if request.JobVersion != 0 {
		job, err = e.store.GetJobVersion(txContext, job.ID, request.JobVersion)
		if err != nil {
			return nil, jobstore.NewJobStoreError(err.Error())
		}
	}

	defer txContext.Rollback() //nolint:errcheck

	switch job.State.StateType {
	case models.JobStateTypePending, models.JobStateTypeQueued, models.JobStateTypeUndefined:
		return nil, bacerrors.Newf("cannot rerun job in state %s", job.State.StateType)
	default:
		// continue
	}

	if err = e.store.UpdateJob(txContext, job); err != nil {
		return nil, err
	}

	if err = e.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
		JobID: job.ID,
		Condition: jobstore.UpdateJobCondition{
			UnexpectedStates: []models.JobStateType{
				models.JobStateTypePending,
				models.JobStateTypeQueued,
				models.JobStateTypeUndefined,
			},
		},
		NewState: models.JobStateTypePending,
		Message:  "job rerun",
	}); err != nil {
		return nil, err
	}

	if err = e.store.AddJobHistory(txContext, job.ID, job.Version+1, JobRerunEvent("job rerun")); err != nil {
		return nil, err
	}

	// enqueue evaluation to allow the scheduler to stop existing executions and start new ones
	now := time.Now().UTC().UnixNano()
	eval := &models.Evaluation{
		ID:          uuid.NewString(),
		JobID:       job.ID,
		TriggeredBy: models.EvalTriggerJobRerun,
		Type:        job.Type,
		Status:      models.EvalStatusPending,
		CreateTime:  now,
		ModifyTime:  now,
	}

	// an evaluation will be created similar to job creation
	if err = e.store.CreateEvaluation(txContext, *eval); err != nil {
		return &RerunJobResponse{}, err
	}

	if err = txContext.Commit(); err != nil {
		return &RerunJobResponse{}, err
	}

	return &RerunJobResponse{
		JobID:        job.ID,
		JobVersion:   job.Version + 1,
		EvaluationID: eval.ID,
		Warnings:     nil,
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
	job, err := e.store.GetJobByIDOrName(ctx, request.JobID, request.Namespace)
	if err != nil {
		return nil, err
	}

	executions, err := e.store.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID:          job.ID,
		JobVersion:     request.JobVersion,
		AllJobVersions: request.AllJobVersions,
	})
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return nil, fmt.Errorf("no executions found for job %s with ID %s, and version %d",
			job.Name,
			job.ID,
			request.JobVersion,
		)
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
		return nil, fmt.Errorf(
			"unable to find execution %s in job %s and job version %d",
			request.ExecutionID,
			job.ID,
			job.Version,
		)
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
	job, err := e.store.GetJobByIDOrName(ctx, request.JobID, request.Namespace)
	if err != nil {
		return GetResultsResponse{}, err
	}

	if job.Type != models.JobTypeBatch && job.Type != models.JobTypeOps {
		return GetResultsResponse{}, fmt.Errorf("job type %s does not support results", job.Type)
	}

	executions, err := e.store.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID:      job.ID,
		JobVersion: job.Version,
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
