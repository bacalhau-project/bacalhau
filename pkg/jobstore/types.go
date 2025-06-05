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
	Namespace string `json:"namespace"`
	// The version of the job to query history for. Takes precedence over LatestJobVersion.
	JobVersion uint64 `json:"job_version"`
	// The latest version of the job. Used for interpreting the job history internally.
	LatestJobVersion uint64 `json:"latest_job_version"`
	// If true, all job versions will be returned, otherwise only the latest job version.
	// This is mutually exclusive with JobVersion, where the latter takes precedence if both are set.
	AllJobVersions        bool   `json:"all_job_versions"`
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

	// GetJobByName returns a job, identified by name and namespace, or an error if
	// it does not exist.
	GetJobByName(ctx context.Context, name, namespace string) (models.Job, error)

	// GetJobByIDOrName returns a job, identified by id, or name and namespace, or an error if
	// it does not exist.
	GetJobByIDOrName(ctx context.Context, idOrName, namespace string) (models.Job, error)

	GetJobVersion(ctx context.Context, jobID string, version uint64) (job models.Job, err error)

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

	// UpdateJob will update an existing job in the store.
	// Only specific fields will be updated, and the job must exist.
	UpdateJob(ctx context.Context, j models.Job) error

	GetJobVersions(ctx context.Context, jobID string) (versions []models.Job, err error)

	// GetExecutions retrieves all executions for the specified job.
	GetExecutions(ctx context.Context, options GetExecutionsOptions) ([]models.Execution, error)

	// UpdateJobState updates the state for the job identified in the
	// [UpdateJobStateRequest].
	UpdateJobState(ctx context.Context, request UpdateJobStateRequest) error

	// AddJobHistory adds a new history entry for the specified job
	AddJobHistory(ctx context.Context, jobID string, jobVersion uint64, events ...models.Event) error

	// CreateExecution creates a new execution
	CreateExecution(ctx context.Context, execution models.Execution) error

	// UpdateExecution updates the execution state according to the values
	// within [UpdateExecutionRequest].
	UpdateExecution(ctx context.Context, request UpdateExecutionRequest) error

	// AddExecutionHistory adds a new history entry for the specified execution
	AddExecutionHistory(ctx context.Context, jobID string, jobVersion uint64, executionID string, events ...models.Event) error

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
	Events      []*models.Event
}

type UpdateJobCondition struct {
	ExpectedState    models.JobStateType
	UnexpectedStates []models.JobStateType
	ExpectedRevision uint64
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
	ExpectedStates        []models.ExecutionStateType
	ExpectedDesiredStates []models.ExecutionDesiredStateType
	ExpectedRevision      uint64
	UnexpectedStates      []models.ExecutionStateType
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
			return NewErrInvalidExecutionState(
				execution.ID, execution.ComputeState.StateType, condition.ExpectedStates...)
		}
	}

	if len(condition.ExpectedDesiredStates) > 0 {
		validState := false
		for _, s := range condition.ExpectedDesiredStates {
			if s == execution.DesiredState.StateType {
				validState = true
				break
			}
		}
		if !validState {
			return NewErrInvalidExecutionDesiredState(
				execution.ID, execution.DesiredState.StateType, condition.ExpectedDesiredStates...)
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
	JobID                   string   `json:"job_id"`
	JobVersion              uint64   `json:"job_version"`
	AllJobVersions          bool     `json:"all_job_versions"`
	CurrentLatestJobVersion uint64   `json:"current_latest_job_version"`
	Namespace               string   `json:"namespace"`
	IncludeJob              bool     `json:"include_job"`
	OrderBy                 string   `json:"order_by"`
	Reverse                 bool     `json:"reverse"`
	Limit                   int      `json:"limit"`
	NodeIDs                 []string `json:"node_ids,omitempty"`         // Filter by one or multiple nodes
	InProgressOnly          bool     `json:"in_progress_only,omitempty"` // Filter to non-terminal executions only
}

// Validate checks if the options are valid
// - JobID, NodeIDs or InProgressOnly must be set
// - If JobVersion is set, AllJobVersions must be false
// - if JobID is not set, then JobVersion cannot be set
func (opts GetExecutionsOptions) Validate() error {
	if opts.JobID == "" && len(opts.NodeIDs) == 0 && !opts.InProgressOnly {
		return NewBadRequestError("bad GetExecutions request: JobID, NodeIDs or InProgressOnly must be set")
	}

	if opts.JobVersion != 0 && opts.AllJobVersions {
		return NewBadRequestError("bad GetExecutions request: JobVersion cannot be set when AllJobVersions is true")
	}

	if opts.JobVersion > 0 && opts.JobID == "" {
		return NewBadRequestError("bad GetExecutions request: JobVersion cannot be set without JobID")
	}
	return nil
}
