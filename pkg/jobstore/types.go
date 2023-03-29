package jobstore

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type JobQuery struct {
	ID          string              `json:"id"`
	ClientID    string              `json:"clientID"`
	IncludeTags []model.IncludedTag `json:"include_tags"`
	ExcludeTags []model.ExcludedTag `json:"exclude_tags"`
	Limit       int                 `json:"limit"`
	Offset      int                 `json:"offset"`
	ReturnAll   bool                `json:"return_all"`
	SortBy      string              `json:"sort_by"`
	SortReverse bool                `json:"sort_reverse"`
}

// A Store will persist jobs and their state to the underlying storage.
// It also gives an efficient way to retrieve jobs using queries.
type Store interface {
	GetJob(ctx context.Context, id string) (model.Job, error)
	GetJobs(ctx context.Context, query JobQuery) ([]model.Job, error)
	GetJobState(ctx context.Context, jobID string) (model.JobState, error)
	GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error)
	GetJobHistory(ctx context.Context, jobID string, since time.Time) ([]model.JobHistory, error)
	GetJobsCount(ctx context.Context, query JobQuery) (int, error)
	CreateJob(ctx context.Context, j model.Job) error
	// UpdateJobState updates the Job state
	UpdateJobState(ctx context.Context, request UpdateJobStateRequest) error
	// CreateExecution creates a new execution for a given job
	CreateExecution(ctx context.Context, execution model.ExecutionState) error
	// UpdateExecution updates the Job state
	UpdateExecution(ctx context.Context, request UpdateExecutionRequest) error
}

type UpdateJobStateRequest struct {
	JobID     string
	Condition UpdateJobCondition
	NewState  model.JobStateType
	Comment   string
}

type UpdateExecutionRequest struct {
	ExecutionID model.ExecutionID
	Condition   UpdateExecutionCondition
	NewValues   model.ExecutionState
	Comment     string
}

type UpdateJobCondition struct {
	ExpectedState    model.JobStateType
	UnexpectedStates []model.JobStateType
	ExpectedVersion  int
}

// Validate checks if the condition matches the given job
func (condition UpdateJobCondition) Validate(jobState model.JobState) error {
	if condition.ExpectedState != model.JobStateNew && condition.ExpectedState != jobState.State {
		return NewErrInvalidJobState(jobState.JobID, jobState.State, condition.ExpectedState)
	}
	if condition.ExpectedVersion != 0 && condition.ExpectedVersion != jobState.Version {
		return NewErrInvalidJobVersion(jobState.JobID, jobState.Version, condition.ExpectedVersion)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == jobState.State {
				return NewErrInvalidJobState(jobState.JobID, jobState.State, model.JobStateNew)
			}
		}
	}
	return nil
}

type UpdateExecutionCondition struct {
	ExpectedState    model.ExecutionStateType
	ExpectedVersion  int
	UnexpectedStates []model.ExecutionStateType
}

// Validate checks if the condition matches the given execution
func (condition UpdateExecutionCondition) Validate(execution model.ExecutionState) error {
	if condition.ExpectedState != model.ExecutionStateNew && condition.ExpectedState != execution.State {
		return NewErrInvalidExecutionState(execution.ID(), execution.State, condition.ExpectedState)
	}
	if condition.ExpectedVersion != 0 && condition.ExpectedVersion != execution.Version {
		return NewErrInvalidExecutionVersion(execution.ID(), execution.Version, condition.ExpectedVersion)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == execution.State {
				return NewErrInvalidExecutionState(execution.ID(), execution.State, model.ExecutionStateNew)
			}
		}
	}
	return nil
}
