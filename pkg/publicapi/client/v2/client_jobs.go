package client

import "github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"

const jobsPath = "/api/v1/orchestrator/jobs"

type Jobs struct {
	client *Client
}

// Jobs returns a handle on the jobs endpoints.
func (c *Client) Jobs() *Jobs {
	return &Jobs{client: c}
}

// Put is used to submit a new job to the cluster, or update an existing job with matching ID.
func (j *Jobs) Put(r *apimodels.PutJobRequest) (*apimodels.PutJobResponse, error) {
	var resp apimodels.PutJobResponse
	if err := j.client.put(jobsPath, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get is used to get a job by ID.
func (j *Jobs) Get(r *apimodels.GetJobRequest) (*apimodels.GetJobResponse, error) {
	var resp apimodels.GetJobResponse
	if err := j.client.get(jobsPath+"/"+r.JobID, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List is used to list all jobs in the cluster.
func (j *Jobs) List(r *apimodels.ListJobsRequest) (*apimodels.ListJobsResponse, error) {
	var resp apimodels.ListJobsResponse
	if err := j.client.list(jobsPath, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// History returns history events for a job.
func (j *Jobs) History(r *apimodels.ListJobHistoryRequest) (*apimodels.ListJobHistoryResponse, error) {
	var resp apimodels.ListJobHistoryResponse
	if err := j.client.list(jobsPath+"/"+r.JobID+"/history", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Executions returns executions for a job.
func (j *Jobs) Executions(r *apimodels.ListJobExecutionsRequest) (*apimodels.ListJobExecutionsResponse, error) {
	var resp apimodels.ListJobExecutionsResponse
	if err := j.client.list(jobsPath+"/"+r.JobID+"/executions", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Results returns results for a job.
func (j *Jobs) Results(r *apimodels.ListJobResultsRequest) (*apimodels.ListJobResultsResponse, error) {
	var resp apimodels.ListJobResultsResponse
	if err := j.client.list(jobsPath+"/"+r.JobID+"/results", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Stop is used to stop a job by ID.
func (j *Jobs) Stop(r *apimodels.StopJobRequest) (*apimodels.StopJobResponse, error) {
	var resp apimodels.StopJobResponse
	if err := j.client.delete(jobsPath+"/"+r.JobID, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
