package apimodels

import (
	"errors"

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
	JobID        string `json:"JobID"`
	EvaluationID string `json:"EvaluationID"`
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
	Labels []*labels.Requirement `query:"-"` // don't auto bind as it requires special handling
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
	JobID string `query:"-"`
	Since int64  `query:"since" validate:"min=0"`
	Type  string `query:"type" validate:"omitempty,oneof=all job execution"`
}

type ListJobHistoryResponse struct {
	BaseListResponse
	History []*models.JobHistory
}

type ListJobExecutionsRequest struct {
	BaseListRequest
	JobID string `query:"job_id" validate:"required"`
}

type ListJobExecutionsResponse struct {
	BaseListResponse
	ExecutionIDs []string
}

type ListJobEvaluationsRequest struct {
	BaseListRequest
	JobID string `query:"job_id" validate:"required"`
}

type ListJobEvaluationsResponse struct {
	BaseListResponse
	EvaluationIDs []string
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

type DescribeJobRequest struct {
	BaseGetRequest
	JobID string
}

type DescribeJobResponse struct {
	BaseGetResponse
}

type SummarizeJobRequest struct {
	BaseGetRequest
	JobID string
}

type SummarizeJobResponse struct {
	BaseGetResponse
}
