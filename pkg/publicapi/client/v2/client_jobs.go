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

// Evaluations returns evaluations for a job.
func (j *Jobs) Evaluations(r *apimodels.ListJobEvaluationsRequest) (*apimodels.ListJobEvaluationsResponse, error) {
	var resp apimodels.ListJobEvaluationsResponse
	if err := j.client.list(jobsPath+"/"+r.JobID+"/evaluations", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Stop is used to stop a job by ID.
func (j *Jobs) Stop(r *apimodels.StopJobRequest) (*apimodels.StopJobResponse, error) {
	var resp apimodels.StopJobResponse
	if err := j.client.delete(jobsPath, r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Describe is used to describe a job by ID.
func (j *Jobs) Describe(r *apimodels.DescribeJobRequest) (*apimodels.DescribeJobResponse, error) {
	var resp apimodels.DescribeJobResponse
	if err := j.client.get(jobsPath+"/"+r.JobID+"description", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Summarize is used to summarize a job by ID.
func (j *Jobs) Summarize(r *apimodels.SummarizeJobRequest) (*apimodels.SummarizeJobResponse, error) {
	var resp apimodels.SummarizeJobResponse
	if err := j.client.get(jobsPath+"/"+r.JobID+"summary", r, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
