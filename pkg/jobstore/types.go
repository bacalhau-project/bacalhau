//go:generate mockgen --source types.go --destination mocks.go --package jobstore
package jobstore

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	// Watch returns a channel from which the caller can read specific events
	// as they are transmitted. When called the combination of parameters
	// will determine which events are sent.  Both the StoreWatcherType and
	// StoreEventType parameters can be a bitmask of entries, so to listen
	// for Create and Delete events for Jobs and Executions you would set
	//   types = JobWatcher | ExecutionWatcher
	//   events = CreateEvent | DeleteEvent
	//
	// The structure sent down the channel when one of these events occurs
	// will contain a timestamp, but also the StoreWatcherType and
	// StoreEventType that triggered the event. A json encoded `[]byte`
	// of the related object will also be included in the [WatchEvent].
	Watch(ctx context.Context, types StoreWatcherType, events StoreEventType) chan WatchEvent

	// GetJob returns a job, identified by the id parameter, or an error if
	// it does not exist.
	GetJob(ctx context.Context, id string) (model.Job, error)

	// GetJobs retrieves a slice of jobs defined by the contents of the
	// [JobQuery]. If it fails, it will return an error
	GetJobs(ctx context.Context, query JobQuery) ([]model.Job, error)

	// GetJobState retrieves the current state for the specified job
	GetJobState(ctx context.Context, jobID string) (model.JobState, error)

	// GetInProgressJobs retrieves all jobs that have a state that can be
	// considered, 'in progress'.  Each job returned is paired with its
	// state in a [model.JobWithInfo]. Failure generates an error.
	GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error)

	// GetJobHistory retrieves the history for the specified job.  The
	// history returned is filtered by the contents of the provided
	// [JobHistoryFilterOptions].
	GetJobHistory(ctx context.Context, jobID string, options JobHistoryFilterOptions) ([]model.JobHistory, error)

	// CreateJob will create a new job and persist it in the store.
	CreateJob(ctx context.Context, j model.Job) error

	// UpdateJobState updates the state for the job identified in the
	// [UpdateJobStateRequest].
	UpdateJobState(ctx context.Context, request UpdateJobStateRequest) error

	// CreateExecution creates a new execution
	CreateExecution(ctx context.Context, execution model.ExecutionState) error

	// UpdateExecution updates the execution state according to the values
	// within [UpdateExecutionRequest].
	UpdateExecution(ctx context.Context, request UpdateExecutionRequest) error

	// DeleteJob removes all trace of the provided job from storage
	DeleteJob(ctx context.Context, jobID string) error

	// CreateEvaluation creates a new evaluation
	CreateEvaluation(ctx context.Context, eval models.Evaluation) error

	// GetEvaluation retrieves the specified evaluation
	GetEvaluation(ctx context.Context, id string) (models.Evaluation, error)

	// DeleteEvaluation deletes the specified evaluation
	DeleteEvaluation(ctx context.Context, id string) error

	// Close provides an interface to cleanup any resources in use when the
	// store is no longer required
	Close(ctx context.Context) error
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
	if !condition.ExpectedState.IsUndefined() && condition.ExpectedState != jobState.State {
		return NewErrInvalidJobState(jobState.JobID, jobState.State, condition.ExpectedState)
	}
	if condition.ExpectedVersion != 0 && condition.ExpectedVersion != jobState.Version {
		return NewErrInvalidJobVersion(jobState.JobID, jobState.Version, condition.ExpectedVersion)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == jobState.State {
				return NewErrInvalidJobState(jobState.JobID, jobState.State, model.JobStateUndefined)
			}
		}
	}
	return nil
}

type UpdateExecutionCondition struct {
	ExpectedStates   []model.ExecutionStateType
	ExpectedVersion  int
	UnexpectedStates []model.ExecutionStateType
}

// Validate checks if the condition matches the given execution
func (condition UpdateExecutionCondition) Validate(execution model.ExecutionState) error {
	if len(condition.ExpectedStates) > 0 {
		validState := false
		for _, s := range condition.ExpectedStates {
			if s == execution.State {
				validState = true
				break
			}
		}
		if !validState {
			return NewErrInvalidExecutionState(execution.ID(), execution.State, condition.ExpectedStates...)
		}
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

type JobHistoryFilterOptions struct {
	Since                 int64 `json:"since"`
	ExcludeExecutionLevel bool  `json:"exclude_execution_level"`
	ExcludeJobLevel       bool  `json:"exclude_job_level"`
}
