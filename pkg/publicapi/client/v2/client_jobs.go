package client

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

const jobsPath = "/api/v1/orchestrator/jobs"

type Jobs struct {
	client *Client
}

// Jobs returns a handle on the jobs endpoints.
func (c *Client) Jobs() *Jobs {
	return &Jobs{client: c}
}

// Put is used to submit a new job to the cluster, or update an existing job with matching ID.
func (j *Jobs) Put(ctx context.Context, r *apimodels.PutJobRequest) (*apimodels.PutJobResponse, error) {
	var resp apimodels.PutJobResponse
	if err := j.client.put(ctx, jobsPath, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get is used to get a job by ID.
func (j *Jobs) Get(ctx context.Context, r *apimodels.GetJobRequest) (*apimodels.GetJobResponse, error) {
	var resp apimodels.GetJobResponse
	if err := j.client.get(ctx, jobsPath+"/"+r.JobID, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List is used to list all jobs in the cluster.
func (j *Jobs) List(ctx context.Context, r *apimodels.ListJobsRequest) (*apimodels.ListJobsResponse, error) {
	var resp apimodels.ListJobsResponse
	if err := j.client.list(ctx, jobsPath, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// History returns history events for a job.
func (j *Jobs) History(ctx context.Context, r *apimodels.ListJobHistoryRequest) (*apimodels.ListJobHistoryResponse, error) {
	var resp apimodels.ListJobHistoryResponse
	if err := j.client.list(ctx, jobsPath+"/"+r.JobID+"/history", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Executions returns executions for a job.
func (j *Jobs) Executions(ctx context.Context, r *apimodels.ListJobExecutionsRequest) (*apimodels.ListJobExecutionsResponse,
	error) {
	var resp apimodels.ListJobExecutionsResponse
	if err := j.client.list(ctx, jobsPath+"/"+r.JobID+"/executions", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Results returns results for a job.
func (j *Jobs) Results(ctx context.Context, r *apimodels.ListJobResultsRequest) (*apimodels.ListJobResultsResponse, error) {
	var resp apimodels.ListJobResultsResponse
	if err := j.client.list(ctx, jobsPath+"/"+r.JobID+"/results", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Stop is used to stop a job by ID.
func (j *Jobs) Stop(ctx context.Context, r *apimodels.StopJobRequest) (*apimodels.StopJobResponse, error) {
	var resp apimodels.StopJobResponse
	if err := j.client.delete(ctx, jobsPath+"/"+r.JobID, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logs returns a stream of logs for a given job/execution.
func (j *Jobs) Logs(ctx context.Context, r *apimodels.GetLogsRequest) (<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	return webSocketDialer[models.ExecutionLog](ctx, j.client, jobsPath+"/"+r.JobID+"/logs", r)
}
