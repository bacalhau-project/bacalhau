package client

import (
	"context"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

const jobsPath = "/api/v1/orchestrator/jobs"

type Jobs struct {
	client Client
}

// Put is used to submit a new job to the cluster, or update an existing job with matching ID.
func (j *Jobs) Put(ctx context.Context, r *apimodels.PutJobRequest) (*apimodels.PutJobResponse, error) {
	var resp apimodels.PutJobResponse
	if err := j.client.Put(ctx, jobsPath, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Diff is used to diff the given spec with the latest Job spec in the cluster.
func (j *Jobs) Diff(ctx context.Context, r *apimodels.DiffJobRequest) (*apimodels.DiffJobResponse, error) {
	var resp apimodels.DiffJobResponse
	if err := j.client.Put(ctx, jobsPath+"/diff", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get is used to get a job by ID or Name.
func (j *Jobs) Get(ctx context.Context, r *apimodels.GetJobRequest) (*apimodels.GetJobResponse, error) {
	var resp apimodels.GetJobResponse
	if err := j.client.Get(ctx, jobsPath+"/"+url.PathEscape(r.JobIDOrName), r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List is used to list all jobs in the cluster.
func (j *Jobs) List(ctx context.Context, r *apimodels.ListJobsRequest) (*apimodels.ListJobsResponse, error) {
	var resp apimodels.ListJobsResponse
	if err := j.client.List(ctx, jobsPath, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// History returns history events for a job.
func (j *Jobs) History(ctx context.Context, r *apimodels.ListJobHistoryRequest) (*apimodels.ListJobHistoryResponse, error) {
	var resp apimodels.ListJobHistoryResponse
	if err := j.client.List(ctx, jobsPath+"/"+url.PathEscape(r.JobIDOrName)+"/history", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Executions returns executions for a job.
func (j *Jobs) Executions(ctx context.Context, r *apimodels.ListJobExecutionsRequest) (*apimodels.ListJobExecutionsResponse,
	error) {
	var resp apimodels.ListJobExecutionsResponse
	if err := j.client.List(ctx, jobsPath+"/"+url.PathEscape(r.JobIDOrName)+"/executions", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Versions returns the versions of a job.
func (j *Jobs) Versions(
	ctx context.Context,
	r *apimodels.ListJobVersionsRequest,
) (
	*apimodels.ListJobVersionsResponse,
	error,
) {
	var resp apimodels.ListJobVersionsResponse
	if err := j.client.List(ctx, jobsPath+"/"+url.PathEscape(r.JobIDOrName)+"/versions", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Results returns results for a job.
func (j *Jobs) Results(ctx context.Context, r *apimodels.ListJobResultsRequest) (*apimodels.ListJobResultsResponse, error) {
	var resp apimodels.ListJobResultsResponse
	if err := j.client.List(ctx, jobsPath+"/"+url.PathEscape(r.JobID)+"/results", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Stop is used to stop a job by ID.
func (j *Jobs) Stop(ctx context.Context, r *apimodels.StopJobRequest) (*apimodels.StopJobResponse, error) {
	var resp apimodels.StopJobResponse
	if err := j.client.Delete(ctx, jobsPath+"/"+url.PathEscape(r.JobID), r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Rerun is used to rerun a job by ID or Name.
func (j *Jobs) Rerun(ctx context.Context, r *apimodels.RerunJobRequest) (*apimodels.RerunJobResponse, error) {
	var resp apimodels.RerunJobResponse
	if err := j.client.Put(ctx, jobsPath+"/"+url.PathEscape(r.JobIDOrName)+"/rerun", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logs returns a stream of logs for a given job/execution.
func (j *Jobs) Logs(ctx context.Context, r *apimodels.GetLogsRequest) (<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	return DialAsyncResult[*apimodels.GetLogsRequest, models.ExecutionLog](ctx, j.client, jobsPath+"/"+r.JobID+"/logs", r)
}
