//go:generate mockgen --source types.go --destination mocks.go --package jobstore
package jobstore

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobQuery struct {
	Namespace string

	// IncludeTags and ExcludeTags are used primarily by the requester's list API.
	// In the orchestrator API, we insted use the Selector field to filter jobs.
	IncludeTags []string
	ExcludeTags []string
	Limit       uint32
	Offset      uint64
	ReturnAll   bool
	SortBy      string
	SortReverse bool
	Selector    labels.Selector
}

type JobQueryResponse struct {
	Jobs       []models.Job
	Offset     uint64 // Offset into the filtered results of the first returned record
	Limit      uint32 // The number of records to return, 0 means all
	NextOffset uint64 // Offset + Limit of the next page of results, 0 means no more results
}

type JobHistoryQuery struct {
	Since                 int64  `json:"since"`
	Limit                 uint32 `json:"limit"`
	ExcludeExecutionLevel bool   `json:"exclude_execution_level"`
	ExcludeJobLevel       bool   `json:"exclude_job_level"`
	ExecutionID           string `json:"execution_id"`
	NextToken             string `json:"next_token"`
}

type JobHistoryQueryResponse struct {
	JobHistory []models.JobHistory
	Offset     uint32
	NextToken  string
}

// TxContext is a transactional context that can be used to commit or rollback
type TxContext interface {
	context.Context
	Commit() error
	Rollback() error
}

// A Store will persist jobs and their state to the underlying storage.
// It also gives an efficient way to retrieve jobs using queries.
type Store interface {
	// BeginTx starts a new transaction and returns a transactional context
	BeginTx(ctx context.Context) (TxContext, error)

	// GetJob returns a job, identified by the id parameter, or an error if
	// it does not exist.
	GetJob(ctx context.Context, id string) (models.Job, error)

	// GetJobs retrieves a slice of jobs defined by the contents of the
	// [JobQuery]. If it fails, it will return an error
	GetJobs(ctx context.Context, query JobQuery) (*JobQueryResponse, error)

	// GetInProgressJobs retrieves all jobs that have a state that can be
	// considered, 'in progress'. Failure generates an error. If the jobType
	// is provided, only active jobs of that type will be returned.
	GetInProgressJobs(ctx context.Context, jobType string) ([]models.Job, error)

	// GetJobHistory retrieves the history for the specified job.  The
	// history returned is filtered by the contents of the provided
	// [JobHistoryFilterOptions].
	GetJobHistory(ctx context.Context, jobID string, options JobHistoryQuery) (*JobHistoryQueryResponse, error)

	// CreateJob will create a new job and persist it in the store.
	CreateJob(ctx context.Context, j models.Job) error

	// GetExecutions retrieves all executions for the specified job.
	GetExecutions(ctx context.Context, options GetExecutionsOptions) ([]models.Execution, error)

	// UpdateJobState updates the state for the job identified in the
	// [UpdateJobStateRequest].
	UpdateJobState(ctx context.Context, request UpdateJobStateRequest) error

	// AddJobHistory adds a new history entry for the specified job
	AddJobHistory(ctx context.Context, jobID string, events ...models.Event) error

	// CreateExecution creates a new execution
	CreateExecution(ctx context.Context, execution models.Execution) error

	// UpdateExecution updates the execution state according to the values
	// within [UpdateExecutionRequest].
	UpdateExecution(ctx context.Context, request UpdateExecutionRequest) error

	// AddExecutionHistory adds a new history entry for the specified execution
	AddExecutionHistory(ctx context.Context, jobID, executionID string, events ...models.Event) error

	// DeleteJob removes all trace of the provided job from storage
	DeleteJob(ctx context.Context, jobID string) error

	// CreateEvaluation creates a new evaluation
	CreateEvaluation(ctx context.Context, eval models.Evaluation) error

	// GetEvaluation retrieves the specified evaluation
	GetEvaluation(ctx context.Context, id string) (models.Evaluation, error)

	// DeleteEvaluation deletes the specified evaluation
	DeleteEvaluation(ctx context.Context, id string) error

	// GetEventStore returns the event store for the execution store
	GetEventStore() watcher.EventStore

	// Close provides an interface to cleanup any resources in use when the
	// store is no longer required
	Close(ctx context.Context) error
}

type UpdateJobStateRequest struct {
	JobID     string
	Condition UpdateJobCondition
	NewState  models.JobStateType
	Message   string
}

type UpdateExecutionRequest struct {
	ExecutionID string
	Condition   UpdateExecutionCondition
	NewValues   models.Execution
}

type UpdateJobCondition struct {
	ExpectedState    models.JobStateType
	UnexpectedStates []models.JobStateType
	ExpectedRevision uint64
}

type ExecutionUpsert struct {
	Current  *models.Execution
	Previous *models.Execution
}

// Validate checks if the condition matches the given job
func (condition UpdateJobCondition) Validate(job models.Job) error {
	if !condition.ExpectedState.IsUndefined() && condition.ExpectedState != job.State.StateType {
		return NewErrInvalidJobState(job.ID, job.State.StateType, condition.ExpectedState)
	}
	if condition.ExpectedRevision != 0 && condition.ExpectedRevision != job.Revision {
		return NewErrInvalidJobVersion(job.ID, job.Revision, condition.ExpectedRevision)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == job.State.StateType {
				return NewErrInvalidJobState(job.ID, job.State.StateType, models.JobStateTypeUndefined)
			}
		}
	}
	return nil
}

type UpdateExecutionCondition struct {
	ExpectedStates   []models.ExecutionStateType
	ExpectedRevision uint64
	UnexpectedStates []models.ExecutionStateType
}

// Validate checks if the condition matches the given execution
func (condition UpdateExecutionCondition) Validate(execution models.Execution) error {
	if len(condition.ExpectedStates) > 0 {
		validState := false
		for _, s := range condition.ExpectedStates {
			if s == execution.ComputeState.StateType {
				validState = true
				break
			}
		}
		if !validState {
			return NewErrInvalidExecutionState(execution.ID, execution.ComputeState.StateType, condition.ExpectedStates...)
		}
	}

	if condition.ExpectedRevision != 0 && condition.ExpectedRevision != execution.Revision {
		return NewErrInvalidExecutionVersion(execution.ID, execution.Revision, condition.ExpectedRevision)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == execution.ComputeState.StateType {
				return NewErrInvalidExecutionState(execution.ID, execution.ComputeState.StateType)
			}
		}
	}
	return nil
}

type GetExecutionsOptions struct {
	JobID      string `json:"job_id"`
	IncludeJob bool   `json:"include_job"`
	OrderBy    string `json:"order_by"`
	Reverse    bool   `json:"reverse"`
	Limit      int    `json:"limit"`
}
