package inmemory

import (
	"context"
	"sort"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/imdario/mergo"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

const newJobComment = "Job created"

type InMemoryJobStore struct {
	// we keep pointers to these things because we will update them partially
	jobs       map[string]model.Job
	states     map[string]model.JobState
	history    map[string][]model.JobHistory
	inprogress map[string]struct{}
	mtx        sync.RWMutex
}

func NewInMemoryJobStore() *InMemoryJobStore {
	res := &InMemoryJobStore{
		jobs:       make(map[string]model.Job),
		states:     make(map[string]model.JobState),
		history:    make(map[string][]model.JobHistory),
		inprogress: make(map[string]struct{}),
	}
	res.mtx.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InMemoryJobStore.mtx",
	})
	return res
}

// Gets a job from the datastore.
//
// Errors:
//
//   - error-job-not-found        		  -- if the job is not found
func (d *InMemoryJobStore) GetJob(_ context.Context, id string) (model.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getJob(id)
}

func (d *InMemoryJobStore) GetJobs(ctx context.Context, query jobstore.JobQuery) ([]model.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	var result []model.Job

	if query.ID != "" {
		j, err := d.getJob(query.ID)
		if err != nil {
			return nil, err
		}
		return []model.Job{j}, nil
	}

	for _, j := range maps.Values(d.jobs) {
		if query.Limit > 0 && len(result) == query.Limit {
			break
		}

		if !query.ReturnAll && query.ClientID != "" && query.ClientID != j.Metadata.ClientID {
			// Job is not for the requesting client, so ignore it.
			continue
		}

		// If we are not using include tags, by default every job is included.
		// If a job is specifically included, that overrides it being excluded.
		included := len(query.IncludeTags) == 0
		for _, tag := range j.Spec.Annotations {
			if slices.Contains(query.IncludeTags, model.IncludedTag(tag)) {
				included = true
				break
			}
			if slices.Contains(query.ExcludeTags, model.ExcludedTag(tag)) {
				included = false
				break
			}
		}

		if !included {
			continue
		}

		result = append(result, j)
	}

	listSorter := func(i, j int) bool {
		switch query.SortBy {
		case "id":
			if query.SortReverse {
				// what does it mean to sort by ID?
				return result[i].Metadata.ID > result[j].Metadata.ID
			} else {
				return result[i].Metadata.ID < result[j].Metadata.ID
			}
		case "created_at":
			if query.SortReverse {
				return result[i].Metadata.CreatedAt.UTC().Unix() > result[j].Metadata.CreatedAt.UTC().Unix()
			} else {
				return result[i].Metadata.CreatedAt.UTC().Unix() < result[j].Metadata.CreatedAt.UTC().Unix()
			}
		default:
			return false
		}
	}
	sort.Slice(result, listSorter)
	return result, nil
}

func (d *InMemoryJobStore) GetJobState(_ context.Context, jobID string) (model.JobState, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	state, ok := d.states[jobID]
	if !ok {
		return model.JobState{}, bacerrors.NewJobNotFound(jobID)
	}
	return state, nil
}

func (d *InMemoryJobStore) GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	var result []model.JobWithInfo
	for id := range d.inprogress {
		result = append(result, model.JobWithInfo{
			Job:   d.jobs[id],
			State: d.states[id],
		})
	}
	return result, nil
}

func (d *InMemoryJobStore) GetJobHistory(_ context.Context, jobID string,
	options jobstore.JobHistoryFilterOptions) ([]model.JobHistory, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	history, ok := d.history[jobID]
	if !ok {
		return nil, jobstore.NewErrJobNotFound(jobID)
	}

	// We want to filter events to only those that happened after the timestamp provided
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

func (d *InMemoryJobStore) CreateJob(_ context.Context, job model.Job) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	existingJob, ok := d.jobs[job.Metadata.ID]
	if ok {
		return jobstore.NewErrJobAlreadyExists(existingJob.Metadata.ID)
	}
	d.jobs[job.Metadata.ID] = job

	// populate job state
	jobState := model.JobState{
		JobID:      job.Metadata.ID,
		State:      model.JobStateNew,
		Version:    1,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}
	d.states[job.Metadata.ID] = jobState
	d.inprogress[job.Metadata.ID] = struct{}{}
	d.appendJobHistory(jobState, model.JobStateNew, newJobComment)
	return nil
}

// helper method to read a single job from memory. This is used by both GetJob and GetJobs.
// It is important that we don't attempt to acquire a lock inside this method to avoid deadlocks since
// the callers are expected to be holding a lock, and golang doesn't support reentrant locks.
func (d *InMemoryJobStore) getJob(id string) (model.Job, error) {
	if len(id) < model.ShortIDLength {
		return model.Job{}, bacerrors.NewJobNotFound(id)
	}

	// support for short job IDs
	if jobutils.ShortID(id) == id {
		// passed in a short id, need to resolve the long id first
		for k := range d.jobs {
			if jobutils.ShortID(k) == id {
				id = k
				break
			}
		}
	}

	j, ok := d.jobs[id]
	if !ok {
		returnError := bacerrors.NewJobNotFound(id)
		return model.Job{}, returnError
	}

	return j, nil
}

func (d *InMemoryJobStore) UpdateJobState(_ context.Context, request jobstore.UpdateJobStateRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// get the existing job state
	jobState, ok := d.states[request.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(request.JobID)
	}

	// check the expected state
	if err := request.Condition.Validate(jobState); err != nil {
		return err
	}
	if jobState.State.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(request.JobID, jobState.State, request.NewState)
	}

	// update the job state
	previousState := jobState.State
	jobState.State = request.NewState
	jobState.Version++
	jobState.UpdateTime = time.Now()
	d.states[request.JobID] = jobState
	if request.NewState.IsTerminal() {
		delete(d.inprogress, request.JobID)
	}
	d.appendJobHistory(jobState, previousState, request.Comment)
	return nil
}

func (d *InMemoryJobStore) CreateExecution(_ context.Context, execution model.ExecutionState) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	jobState, ok := d.states[execution.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}
	for _, e := range jobState.Executions {
		if e.ID() == execution.ID() {
			return jobstore.NewErrExecutionAlreadyExists(execution.ID())
		}
	}
	if execution.CreateTime.IsZero() {
		execution.CreateTime = time.Now()
	}
	if execution.UpdateTime.IsZero() {
		execution.UpdateTime = execution.CreateTime
	}
	if execution.Version == 0 {
		execution.Version = 1
	}
	jobState.Executions = append(jobState.Executions, execution)
	d.states[execution.JobID] = jobState
	d.appendExecutionHistory(execution, model.ExecutionStateNew, "")
	return nil
}

func (d *InMemoryJobStore) UpdateExecution(_ context.Context, request jobstore.UpdateExecutionRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// find the existing execution
	jobState, ok := d.states[request.ExecutionID.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(request.ExecutionID.JobID)
	}
	var existingExecution model.ExecutionState
	executionIndex := -1
	for i, e := range jobState.Executions {
		if e.ID() == request.ExecutionID {
			existingExecution = e
			executionIndex = i
			break
		}
	}
	if executionIndex == -1 {
		return jobstore.NewErrExecutionNotFound(request.ExecutionID)
	}

	// check the expected state
	if err := request.Condition.Validate(existingExecution); err != nil {
		return err
	}
	if existingExecution.State.IsTerminal() {
		return jobstore.NewErrExecutionAlreadyTerminal(request.ExecutionID, existingExecution.State, request.NewValues.State)
	}

	// populate default values
	newExecution := request.NewValues
	if newExecution.CreateTime.IsZero() {
		newExecution.CreateTime = time.Now()
	}
	if newExecution.UpdateTime.IsZero() {
		newExecution.UpdateTime = existingExecution.CreateTime
	}
	if newExecution.Version == 0 {
		newExecution.Version = existingExecution.Version + 1
	}

	err := mergo.Merge(&newExecution, existingExecution)
	if err != nil {
		return err
	}

	// update the execution
	previousState := existingExecution.State
	jobState.Executions[executionIndex] = newExecution
	d.states[newExecution.JobID] = jobState
	d.appendExecutionHistory(newExecution, previousState, request.Comment)
	return nil
}

func (d *InMemoryJobStore) Close(_ context.Context) error {
	return nil
}

func (d *InMemoryJobStore) appendJobHistory(updateJob model.JobState, previousState model.JobStateType, comment string) {
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
	d.history[updateJob.JobID] = append(d.history[updateJob.JobID], historyEntry)
}

func (d *InMemoryJobStore) appendExecutionHistory(updatedExecution model.ExecutionState,
	previousState model.ExecutionStateType, comment string) {
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
	d.history[updatedExecution.JobID] = append(d.history[updatedExecution.JobID], historyEntry)
}

// Static check to ensure that Transport implements Transport:
var _ jobstore.Store = (*InMemoryJobStore)(nil)
