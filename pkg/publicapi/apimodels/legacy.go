package apimodels

import (
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/signatures"
)

type CancelRequest = signatures.SignedRequest[model.JobCancelPayload]

type CancelResponse struct {
	State *model.JobState `json:"state"`
}

type EventsRequest struct {
	ClientID string             `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	JobID    string             `json:"job_id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	Options  EventFilterOptions `json:"filters"` // Records the number of seconds since the unix epoch (UTC)
}

type EventsResponse struct {
	Events []model.JobHistory `json:"events"`
}

type EventFilterOptions = jobstore.JobHistoryFilterOptions

type ListRequest struct {
	JobID       string   `json:"id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	ClientID    string   `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	IncludeTags []string `json:"include_tags" example:"['any-tag']"`
	ExcludeTags []string `json:"exclude_tags" example:"['any-tag']"`
	MaxJobs     int      `json:"max_jobs" example:"10"`
	ReturnAll   bool     `json:"return_all" `
	SortBy      string   `json:"sort_by" example:"created_at"`
	SortReverse bool     `json:"sort_reverse"`
}

type ListResponse struct {
	Jobs []*model.JobWithInfo `json:"jobs"`
}

type ResultsRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	JobID    string `json:"job_id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
}

type ResultsResponse struct {
	Results []model.PublishedResult `json:"results"`
}

type StateRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	JobID    string `json:"job_id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
}

type StateResponse struct {
	State model.JobState `json:"state"`
}

type SubmitRequest = signatures.SignedRequest[model.JobCreatePayload]

type SubmitResponse struct {
	Job *model.Job `json:"job"`
}

type LogRequest = signatures.SignedRequest[model.LogsPayload]
