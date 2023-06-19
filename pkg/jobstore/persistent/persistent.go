package persistent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/distributed"
	"github.com/google/uuid"
	"github.com/imdario/mergo"
)

const newJobComment = "Job created"

type PersistentJobStore struct {
	store objectstore.ObjectStore
}

var ErrNotFound = errors.New("object not found")

const (
	PrefixJobs       = "jobs"
	PrefixActiveJobs = "activejobs"
	PrefixJobState   = "jobstate"
	PrefixJobHistory = "history"
)

// NewPersistentJobStore creates a new JobStore that implements `jobstore.Store` for
// holding Jobs, JobState, JobHistory etc.  These are held under different prefixes:
//
//		Job (model.Job)               -> "jobs"
//	    ActiveJobs (model.Job)		  -> "activejobs"
//		JobHistory (model.JobHistory) -> "history"
//		JobState (model.JobState)     -> "jobstate"
func NewPersistentJobStore(ctx context.Context) (*PersistentJobStore, error) {
	store, err := objectstore.GetImplementation(ctx, objectstore.DistributedImplementation, distributed.WithTestConfig())
	if err != nil {
		return nil, err
	}

	return &PersistentJobStore{
		store: store,
	}, nil
}

func (p *PersistentJobStore) Close(ctx context.Context) error {
	return p.store.Close(ctx)
}

// Gets a job from the datastore.
//
// Errors:
//
//   - error-job-not-found        		  -- if the job is not found
func (p *PersistentJobStore) GetJob(ctx context.Context, id string) (model.Job, error) {
	return p.getJob(ctx, id)
}

// 	if !query.ReturnAll && query.ClientID != "" && query.ClientID != j.Metadata.ClientID {
// 		if slices.Contains(query.IncludeTags, model.IncludedTag(tag)) {
// 		if slices.Contains(query.ExcludeTags, model.ExcludedTag(tag)) {

func (p *PersistentJobStore) GetJobs(ctx context.Context, query jobstore.JobQuery) ([]model.Job, error) {
	if query.ID != "" {
		j, err := p.getJob(ctx, query.ID)
		if err != nil {
			return nil, err
		}
		return []model.Job{j}, nil
	}

	var keys []string

	keys, err := p.store.List(ctx, PrefixJobs)
	if err != nil {
		return nil, err
	}

	if query.Limit > 0 && len(keys) > query.Limit {
		keys = keys[query.Offset:query.Limit]
	}

	var jobs []model.Job
	found, err := p.store.GetBatch(ctx, PrefixJobs, keys, jobs)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}

	// TODO: Requires "does this key appear in the list for clientid?"
	// if !query.ReturnAll && query.ClientID != "" && query.ClientID != j.Metadata.ClientID {
	// 	// Job is not for the requesting client, so ignore it.
	// 	continue
	// }

	// 	// If we are not using include tags, by default every job is included.
	// 	// If a job is specifically included, that overrides it being excluded.
	// 	included := len(query.IncludeTags) == 0
	// 	for _, tag := range j.Spec.Annotations {
	// 		if slices.Contains(query.IncludeTags, model.IncludedTag(tag)) {
	// 			included = true
	// 			break
	// 		}
	// 		if slices.Contains(query.ExcludeTags, model.ExcludedTag(tag)) {
	// 			included = false
	// 			break
	// 		}
	// 	}

	// 	if !included {
	// 		continue
	// 	}

	listSorter := func(i, j int) bool {
		switch query.SortBy {
		case "id":
			if query.SortReverse {
				// what does it mean to sort by ID?
				return jobs[i].Metadata.ID > jobs[j].Metadata.ID
			} else {
				return jobs[i].Metadata.ID < jobs[j].Metadata.ID
			}
		case "created_at":
			if query.SortReverse {
				return jobs[i].Metadata.CreatedAt.UTC().Unix() > jobs[j].Metadata.CreatedAt.UTC().Unix()
			} else {
				return jobs[i].Metadata.CreatedAt.UTC().Unix() < jobs[j].Metadata.CreatedAt.UTC().Unix()
			}
		default:
			return false
		}
	}
	sort.Slice(jobs, listSorter)

	return jobs, nil
}

func (p *PersistentJobStore) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	state := model.JobState{}
	found, err := p.store.Get(ctx, PrefixJobState, jobID, &state)
	if err != nil {
		return state, err
	}

	if !found {
		return state, ErrNotFound
	}
	return state, nil
}

func (p *PersistentJobStore) GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error) {
	// var result []model.JobWithInfo
	// var job model.Job

	// p.store.Get(ctx)

	// for id := range d.inprogress {
	// 	result = append(result, model.JobWithInfo{
	// 		Job:   d.jobs[id],
	// 		State: d.states[id],
	// 	})
	// }
	//return result, nil
	return nil, nil
}

func (p *PersistentJobStore) GetJobHistory(ctx context.Context,
	jobID string, options jobstore.JobHistoryFilterOptions) ([]model.JobHistory, error) {
	// Get partial matches
	prefix := fmt.Sprintf("%s/%s", PrefixJobHistory, jobID)

	keys, err := p.store.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	var history []model.JobHistory
	found, err := p.store.GetBatch(ctx, prefix, keys, &history)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, jobstore.NewErrJobNotFound(jobID)
	}

	// // We want to filter events to only those that happened after the timestamp provided
	sinceTime := options.Since
	eventList := make([]model.JobHistory, 0, len(history))
	for _, event := range history {
		if options.ExcludeExecutionLevel && event.Type == model.JobHistoryTypeExecutionLevel {
			continue
		}

		if options.ExcludeJobLevel && event.Type == model.JobHistoryTypeJobLevel {
			continue
		}

		if event.Time.Unix() >= sinceTime {
			eventList = append(eventList, event)
		}
	}

	history = eventList
	sort.Slice(history, func(i, j int) bool { return history[i].Time.UTC().Before(history[j].Time.UTC()) })
	return history, nil
}

func (p *PersistentJobStore) GetJobsCount(ctx context.Context, query jobstore.JobQuery) (int, error) {
	useQuery := query
	useQuery.Limit = 0
	useQuery.Offset = 0
	jobs, err := p.GetJobs(ctx, useQuery)
	if err != nil {
		return 0, err
	}
	return len(jobs), nil
}

func (p *PersistentJobStore) CreateJob(ctx context.Context, job model.Job) error {
	existingJob, err := p.getJob(ctx, job.Metadata.ID)
	if err == nil {
		return jobstore.NewErrJobAlreadyExists(existingJob.Metadata.ID)
	}

	jobState := model.JobState{
		JobID:      job.Metadata.ID,
		State:      model.JobStateNew,
		Version:    1,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	if err = p.store.Put(ctx, PrefixJobState, jobState.JobID, jobState); err != nil {
		return err
	}

	if err = p.store.Put(ctx, PrefixActiveJobs, job.ID(), &job); err != nil {
		return err
	}

	return p.appendJobHistory(ctx, jobState, model.JobStateNew, newJobComment)
}

// helper method to read a single job from memory. This is used by both GetJob and GetJobs.
// It is important that we don't attempt to acquire a lock inside this method to avoid deadlocks since
// the callers are expected to be holding a lock, and golang doesn't support reentrant locks.
func (p *PersistentJobStore) getJob(ctx context.Context, id string) (model.Job, error) {
	if len(id) < model.ShortIDLength {
		return model.Job{}, bacerrors.NewJobNotFound(id)
	}

	// TODO: For etcd we can do this with a prefix partial match on the short id.

	// // support for short job IDs
	if jobutils.ShortID(id) == id {
		// passed in a short id, need to resolve the long id first
		allJobs, _ := p.store.List(ctx, PrefixJobs)
		for _, k := range allJobs {
			if jobutils.ShortID(k) == id {
				id = k
				break
			}
		}
	}

	var job model.Job

	found, err := p.store.Get(ctx, PrefixJobs, id, &job)
	if err != nil {
		return model.Job{}, err
	}
	if !found {
		return model.Job{}, bacerrors.NewJobNotFound(id)
	}

	return job, nil
}

func (p *PersistentJobStore) UpdateJobState(ctx context.Context, request jobstore.UpdateJobStateRequest) error {
	state, err := p.GetJobState(ctx, request.JobID)
	if err != nil {
		return jobstore.NewErrJobNotFound(request.JobID)
	}

	// // check the expected state
	if err := request.Condition.Validate(state); err != nil {
		return err
	}
	if state.State.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(request.JobID, state.State, request.NewState)
	}

	// // update the job state
	previousState := state.State
	state.State = request.NewState
	state.Version++
	state.UpdateTime = time.Now().UTC()

	err = p.store.Put(ctx, PrefixJobState, state.JobID, &state)
	if err != nil {
		return err
	}

	if request.NewState.IsTerminal() {
		job := model.Job{}
		_, err := p.store.Get(ctx, PrefixActiveJobs, request.JobID, &job)
		if err != nil {
			return err
		}

		err = p.store.Delete(ctx, PrefixActiveJobs, request.JobID, &job)
		if err != nil {
			return err
		}
	}

	return p.appendJobHistory(ctx, state, previousState, request.Comment)
}

// TODO: Want to do this in a trxn ...
func (p *PersistentJobStore) CreateExecution(ctx context.Context, execution model.ExecutionState) error {
	state, err := p.GetJobState(ctx, execution.JobID)
	if err != nil {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}

	for _, e := range state.Executions {
		if e.ID() == execution.ID() {
			return jobstore.NewErrExecutionAlreadyExists(execution.ID())
		}
	}

	if execution.CreateTime.IsZero() {
		execution.CreateTime = time.Now().UTC()
	}
	if execution.UpdateTime.IsZero() {
		execution.UpdateTime = execution.CreateTime
	}
	if execution.Version == 0 {
		execution.Version = 1
	}

	state.Executions = append(state.Executions, execution)
	err = p.store.Put(ctx, PrefixJobState, execution.JobID, &state)
	if err != nil {
		return err
	}

	return p.appendExecutionHistory(ctx, execution, model.ExecutionStateNew, "")
}

func (p *PersistentJobStore) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) error {
	state, err := p.GetJobState(ctx, request.ExecutionID.JobID)
	if err != nil {
		return jobstore.NewErrJobNotFound(request.ExecutionID.JobID)
	}

	var existingExecution model.ExecutionState
	executionIndex := -1

	for i, e := range state.Executions {
		if e.ID() == request.ExecutionID {
			existingExecution = e
			executionIndex = i
			break
		}
	}
	if executionIndex == -1 {
		return jobstore.NewErrExecutionNotFound(request.ExecutionID)
	}

	// // check the expected state
	if err := request.Condition.Validate(existingExecution); err != nil {
		return err
	}
	if existingExecution.State.IsTerminal() {
		return jobstore.NewErrExecutionAlreadyTerminal(request.ExecutionID, existingExecution.State, request.NewValues.State)
	}

	// // populate default values
	newExecution := request.NewValues
	if newExecution.CreateTime.IsZero() {
		newExecution.CreateTime = time.Now().UTC()
	}
	if newExecution.UpdateTime.IsZero() {
		newExecution.UpdateTime = existingExecution.CreateTime
	}
	if newExecution.Version == 0 {
		newExecution.Version = existingExecution.Version + 1
	}

	err = mergo.Merge(&newExecution, existingExecution)
	if err != nil {
		return err
	}

	// // update the execution
	previousState := existingExecution.State
	state.Executions[executionIndex] = newExecution

	err = p.store.Put(ctx, PrefixJobState, request.ExecutionID.JobID, &state)
	if err != nil {
		return err
	}

	return p.appendExecutionHistory(ctx, newExecution, previousState, request.Comment)
}

func (p *PersistentJobStore) appendJobHistory(ctx context.Context,
	updateJob model.JobState, previousState model.JobStateType, comment string) error {
	historyEntry := model.JobHistory{
		Type:  model.JobHistoryTypeJobLevel,
		JobID: updateJob.JobID,
		JobState: &model.StateChange[model.JobStateType]{
			Previous: previousState,
			New:      updateJob.State,
		},
		NewVersion: updateJob.Version,
		Comment:    comment,
		Time:       updateJob.UpdateTime,
	}

	// Generate a key for the history, we are never going to really ask for this item by key
	// without a previous list command, so it doesn't need to make sense.
	key, _ := uuid.NewUUID()
	prefix := fmt.Sprintf("%s/%s", PrefixJobHistory, updateJob.JobID)
	err := p.store.Put(ctx, prefix, key.String(), historyEntry)
	if err != nil {
		return err
	}

	return nil
}

func (p *PersistentJobStore) appendExecutionHistory(ctx context.Context,
	updatedExecution model.ExecutionState, previousState model.ExecutionStateType, comment string) error {
	historyEntry := model.JobHistory{
		Type:             model.JobHistoryTypeExecutionLevel,
		JobID:            updatedExecution.JobID,
		NodeID:           updatedExecution.NodeID,
		ComputeReference: updatedExecution.ComputeReference,
		ExecutionState: &model.StateChange[model.ExecutionStateType]{
			Previous: previousState,
			New:      updatedExecution.State,
		},
		NewVersion: updatedExecution.Version,
		Comment:    comment,
		Time:       updatedExecution.UpdateTime,
	}

	// Generate a key for the history, we are never going to really ask for this item by key
	// without a previous list command, so it doesn't need to make sense.
	key, _ := uuid.NewUUID()
	prefix := fmt.Sprintf("%s/%s", PrefixJobHistory, updatedExecution.JobID)

	err := p.store.Put(ctx, prefix, key.String(), historyEntry)
	if err != nil {
		return err
	}

	return nil
}
