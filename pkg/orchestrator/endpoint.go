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

	// TODO: Implement warnings for the name syntax if it is not DNS compliant
	// Need to check here because a Job ID gets generated. We should be careful what to do here.
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

	// Job Transformer
	// TODO: check what is happening in this transform, for updates especailly.
	if err = e.jobTransformer.Transform(ctx, job); err != nil {
		return nil, err
	}

	isUpdate := false
	evalTriggeredBy := models.EvalTriggerJobRegister
	// Try to see if a job with the same name and namespace exist, if yes, change the JOb ID
	// to the ID if the Job
	existingJob, existingErr := e.store.GetJobByName(ctx, job.Name, job.Namespace)
	if existingErr == nil {
		// This is an update
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
				)
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

	// We need a runtime ID
	// everytime there is a submission of a new job or expelcit update, it will create a new runtime id
	//    and encodes that in the evaluation that an update is needed
	// When the scheduler picks it up and sees that it is an update,
	//    it will cancel all executions that do not have a runtime ID and not similar
	//    and schedules new executions with this runtime same ID
	// We should probably see if there is any evaluation going when submitting update,
	// 	  do not allow update, i.e reject enqueing the evaluation
	//

	// Create or update the job based on whether it's a new job or an update
	if isUpdate {
		// For updates, use the UpdateJob method
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

		// Add job history for the update
		if err = e.store.AddJobHistory(txContext, job.ID, JobUpdatedEvent()); err != nil {
			return nil, err
		}
	} else {
		// For new jobs, use the CreateJob method
		if err = e.store.CreateJob(txContext, *job); err != nil {
			return nil, err
		}
		// Add job history for the new submission
		if err = e.store.AddJobHistory(txContext, job.ID, JobSubmittedEvent()); err != nil {
			return nil, err
		}
	}

	// need to figure out the evaluation
	// the evaluation is just a signal
	// This is the evaluation that will trigger the flow it seems
	// TODO: this type below is used to pick up the needed scheduler
	eval := &models.Evaluation{
		ID:          uuid.NewString(),
		JobID:       job.ID,
		TriggeredBy: evalTriggeredBy,
		Type:        job.Type,
		Status:      models.EvalStatusPending,
		CreateTime:  job.CreateTime,
		ModifyTime:  job.CreateTime,
		IsUpdate:    isUpdate,
		RuntimeID:   uuid.NewString(),
	}

	// maybe we should create an evalution that will update the jobs ?
	// the evalution creation will create an event that notifies the watcher
	// that the evaluation has been created
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

	// ths is where the desired state is added
	// this is only called for stop job not create job
	// update the job state, except if the job is already completed
	// we allow marking a failed job as canceled
	if err = e.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
		JobID: job.ID, // use the job ID from the store in case the request had a short ID, or it had a JobName
		// the condition is that it is not complete
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
			IsUpdate:    false,
			RuntimeID:   uuid.NewString(),
		}

		// an evaluation will be created similar to job creation
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

func (e *BaseEndpoint) RerunJob(ctx context.Context, request *RerunJobRequest) (RerunJobResponse, error) {
	txContext, err := e.store.BeginTx(ctx)
	if err != nil {
		return RerunJobResponse{}, jobstore.NewJobStoreError(err.Error())
	}
	defer txContext.Rollback() //nolint:errcheck

	job, err := e.store.GetJobByIDOrName(txContext, request.JobID, request.Namespace)
	if err != nil {
		return RerunJobResponse{}, err
	}

	switch job.State.StateType {
	case models.JobStateTypePending, models.JobStateTypeQueued, models.JobStateTypeUndefined:
		return RerunJobResponse{}, bacerrors.Newf("cannot rerun job in state %s", job.State.StateType)
	default:
		// continue
	}

	if err = e.store.UpdateJobState(txContext, jobstore.UpdateJobStateRequest{
		JobID: job.ID, // use the job ID from the store in case the request had a short ID, or passed as JobName
		// the condition is that it is not complete
		Condition: jobstore.UpdateJobCondition{
			UnexpectedStates: []models.JobStateType{
				models.JobStateTypePending,
				models.JobStateTypeQueued,
				models.JobStateTypeUndefined,
			},
		},
		NewState: models.JobStateTypePending,
		Message:  request.Reason,
	}); err != nil {
		return RerunJobResponse{}, err
	}

	if err = e.store.AddJobHistory(txContext, job.ID, JobRerunEvent(request.Reason)); err != nil {
		return RerunJobResponse{}, err
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
		IsUpdate:    true,
		RuntimeID:   uuid.NewString(),
	}

	// an evaluation will be created similar to job creation
	if err = e.store.CreateEvaluation(txContext, *eval); err != nil {
		return RerunJobResponse{}, err
	}
	evalID := eval.ID

	if err = txContext.Commit(); err != nil {
		return RerunJobResponse{}, err
	}

	return RerunJobResponse{
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
	job, err := e.store.GetJobByIDOrName(ctx, request.JobID, request.Namespace)
	if err != nil {
		return nil, err
	}

	executions, err := e.store.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: job.ID,
	})
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return nil, fmt.Errorf("no executions found for job %s", job.ID)
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
		return nil, fmt.Errorf("unable to find execution %s in job %s", request.ExecutionID, job.ID)
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
