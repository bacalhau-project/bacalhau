package apimodels

import (
	"errors"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/labels"
)

type PutJobRequest struct {
	BasePutRequest
	Job *models.Job `json:"Job"`
}

// Normalize is used to canonicalize fields in the PutJobRequest.
func (r *PutJobRequest) Normalize() {
	if r.Job != nil {
		r.Job.Normalize()
	}
}

// Validate is used to validate fields in the PutJobRequest.
func (r *PutJobRequest) Validate() error {
	mErr := new(multierror.Error)
	if r.Job == nil {
		mErr = multierror.Append(mErr, errors.New("job is required"))
	}
	jobErrs := r.Job.ValidateSubmission()
	if jobErrs != nil {
		mErr = multierror.Append(mErr, jobErrs)
	}
	return mErr.ErrorOrNil()
}

type PutJobResponse struct {
	BasePutResponse
	JobID        string   `json:"JobID"`
	EvaluationID string   `json:"EvaluationID"`
	Warnings     []string `json:"Warnings"`
}

type GetJobRequest struct {
	BaseGetRequest
	JobID string
}

type GetJobResponse struct {
	BaseGetResponse
	Job *models.Job `json:"Job"`
}

// Normalize is used to33 canonicalize fields in the GetJobResponse.
func (r *GetJobResponse) Normalize() {
	r.BaseGetResponse.Normalize()
	if r.Job != nil {
		r.Job.Normalize()
	}
}

type ListJobsRequest struct {
	BaseListRequest
	Labels []labels.Requirement `query:"-"` // don't auto bind as it requires special handling
}

// ToHTTPRequest is used to convert the request to an HTTP request
func (o *ListJobsRequest) ToHTTPRequest() *HTTPRequest {
	r := o.BaseListRequest.ToHTTPRequest()

	for _, v := range o.Labels {
		r.Params.Add("labels", v.String())
	}
	return r
}

type ListJobsResponse struct {
	BaseListResponse
	Jobs []*models.Job `json:"Jobs"`
}

// Normalize is used to canonicalize fields in the ListJobsResponse.
func (r *ListJobsResponse) Normalize() {
	r.BaseListResponse.Normalize()
	for _, job := range r.Jobs {
		job.Normalize()
	}
}

type ListJobHistoryRequest struct {
	BaseListRequest
	JobID       string `query:"-"`
	Since       int64  `query:"since" validate:"min=0"`
	EventType   string `query:"event_type" validate:"omitempty,oneof=all job execution"`
	ExecutionID string `query:"execution_id" validate:"omitempty"`
	NodeID      string `query:"node_id" validate:"omitempty"`
}

// ToHTTPRequest is used to convert the request to an HTTP request
func (o *ListJobHistoryRequest) ToHTTPRequest() *HTTPRequest {
	r := o.BaseListRequest.ToHTTPRequest()

	if o.Since != 0 {
		r.Params.Set("since", strconv.FormatInt(o.Since, 10))
	}
	if o.EventType != "" {
		r.Params.Set("event_type", o.EventType)
	}
	if o.ExecutionID != "" {
		r.Params.Set("execution_id", o.ExecutionID)
	}
	if o.NodeID != "" {
		r.Params.Set("node_id", o.NodeID)
	}
	return r
}

type ListJobHistoryResponse struct {
	BaseListResponse
	History []*models.JobHistory
}

type ListJobExecutionsRequest struct {
	BaseListRequest
	JobID string `query:"-"`
}

type ListJobExecutionsResponse struct {
	BaseListResponse
	Executions []*models.Execution
}

type ListJobResultsRequest struct {
	BaseListRequest
	JobID string `query:"-"`
}

type ListJobResultsResponse struct {
	BaseListResponse
	Results []*models.SpecConfig
}

type StopJobRequest struct {
	BasePutRequest
	JobID  string `json:"-"`
	Reason string `json:"reason"`
}

type StopJobResponse struct {
	BasePutResponse
	EvaluationID string `json:"EvaluationID"`
}
