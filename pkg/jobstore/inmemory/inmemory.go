package inmemory

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/benbjohnson/clock"
	"github.com/imdario/mergo"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const newJobComment = "Job created"

type InMemoryJobStore struct {
	// jobs is a map of job ID to job
	jobs map[string]models.Job
	// executions is map of execution ID to execution
	executions map[string]models.Execution
	// jobExecutions is a map of job ID to list of execution IDs
	jobExecutions map[string][]string
	// history is a map of job ID to job history
	history map[string][]models.JobHistory
	// inProgress is a set of job IDs that are in progress
	inProgress map[string]struct{}
	// evaluations is a map of evaluation ID to evaluation
	evaluations map[string]models.Evaluation
	watchers    []jobstore.Watcher
	watcherLock sync.Mutex
	mtx         sync.RWMutex
	clock       clock.Clock
}

type Option func(store *(InMemoryJobStore))

func WithClock(clock clock.Clock) Option {
	return func(store *InMemoryJobStore) {
		store.clock = clock
	}
}

func NewInMemoryJobStore(options ...Option) *InMemoryJobStore {
	res := &InMemoryJobStore{
		jobs:          make(map[string]models.Job),
		executions:    make(map[string]models.Execution),
		jobExecutions: make(map[string][]string),
		history:       make(map[string][]models.JobHistory),
		inProgress:    make(map[string]struct{}),
		evaluations:   make(map[string]models.Evaluation),
		watchers:      make([]jobstore.Watcher, 1),
		clock:         clock.New(),
	}
	for _, opt := range options {
		opt(res)
	}

	return res
}

func (d *InMemoryJobStore) Watch(c context.Context, t jobstore.StoreWatcherType, e jobstore.StoreEventType) chan jobstore.WatchEvent {
	w := jobstore.NewWatcher(t, e)

	d.watcherLock.Lock()
	d.watchers = append(d.watchers, *w)
	d.watcherLock.Unlock()

	return w.Channel()
}

func (d *InMemoryJobStore) triggerEvent(t jobstore.StoreWatcherType, e jobstore.StoreEventType, object interface{}) {
	data, _ := json.Marshal(object)

	for _, w := range d.watchers {
		if w.IsWatchingEvent(e) && w.IsWatchingType(t) {
			w.Channel() <- jobstore.WatchEvent{
				Kind:   t,
				Event:  e,
				Object: data,
			}
		}
	}
}

// Gets a job from the datastore.
//
// Errors:
//
//   - error-job-not-found        		  -- if the job is not found
func (d *InMemoryJobStore) GetJob(_ context.Context, id string) (models.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getJob(id)
}

func (d *InMemoryJobStore) GetJobs(ctx context.Context, query jobstore.JobQuery) ([]models.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	var result []models.Job

	if query.ID != "" {
		j, err := d.getJob(query.ID)
		if err != nil {
			return nil, err
		}
		return []models.Job{j}, nil
	}

	for _, j := range maps.Values(d.jobs) {
		if query.Limit > 0 && uint32(len(result)) == query.Limit {
			break
		}

		if !query.ReturnAll && query.Namespace != "" && query.Namespace != j.Namespace {
			// Job is not for the requesting client, so ignore it.
			continue
		}

		// If we are not using include tags, by default every job is included.
		// If a job is specifically excluded, that overrides it being included.
		included := len(query.IncludeTags) == 0
		for tag := range j.Labels {
			if slices.Contains(query.IncludeTags, tag) {
				included = true
			}
			if slices.Contains(query.ExcludeTags, tag) {
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
				return result[i].ID > result[j].ID
			} else {
				return result[i].ID < result[j].ID
			}
		case "created_at":
			if query.SortReverse {
				return result[i].CreateTime > result[j].CreateTime
			} else {
				return result[i].CreateTime < result[j].CreateTime
			}
		default:
			return false
		}
	}
	sort.Slice(result, listSorter)
	return result, nil
}

func (d *InMemoryJobStore) GetExecutions(_ context.Context, jobID string) ([]models.Execution, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	jobID, err := d.reifyJobID(jobID)
	if err != nil {
		return nil, err
	}

	executionIDs, ok := d.jobExecutions[jobID]
	if !ok {
		return nil, bacerrors.NewJobNotFound(jobID)
	}
	result := make([]models.Execution, 0, len(executionIDs))
	for _, id := range executionIDs {
		result = append(result, d.executions[id])
	}
	return result, nil
}

func (d *InMemoryJobStore) GetInProgressJobs(ctx context.Context) ([]models.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	var result []models.Job
	for id := range d.inProgress {
		result = append(result, d.jobs[id])
	}
	return result, nil
}

func (d *InMemoryJobStore) GetJobHistory(_ context.Context, jobID string,
	options jobstore.JobHistoryFilterOptions) ([]models.JobHistory, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	jobID, err := d.reifyJobID(jobID)
	if err != nil {
		return nil, err
	}

	history, ok := d.history[jobID]
	if !ok {
		return nil, jobstore.NewErrJobNotFound(jobID)
	}

	// We want to filter events to only those that happened after the timestamp provided
	sinceTime := options.Since
	eventList := make([]models.JobHistory, 0, len(history))
	for _, event := range history {
		if options.ExcludeExecutionLevel && event.Type == models.JobHistoryTypeExecutionLevel {
			continue
		}

		if options.ExcludeJobLevel && event.Type == models.JobHistoryTypeJobLevel {
			continue
		}

		if options.ExecutionID != "" && strings.HasPrefix(event.ExecutionID, options.ExecutionID) {
			continue
		}

		if options.NodeID != "" && strings.HasPrefix(event.NodeID, options.NodeID) {
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

func (d *InMemoryJobStore) CreateJob(_ context.Context, job models.Job) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	existingJob, ok := d.jobs[job.ID]
	if ok {
		return jobstore.NewErrJobAlreadyExists(existingJob.ID)
	}
	job.State = models.NewJobState(models.JobStateTypePending)
	job.Revision = 1
	job.CreateTime = d.clock.Now().UTC().UnixNano()
	job.ModifyTime = d.clock.Now().UTC().UnixNano()
	job.Normalize()

	err := job.Validate()
	if err != nil {
		return err
	}
	d.jobs[job.ID] = job

	// populate job state
	d.jobExecutions[job.ID] = []string{}
	d.inProgress[job.ID] = struct{}{}
	d.appendJobHistory(job, models.JobStateTypePending, newJobComment)

	d.triggerEvent(jobstore.JobWatcher, jobstore.CreateEvent, job)

	return nil
}

// DeleteJob removes a job from storage
func (d *InMemoryJobStore) DeleteJob(ctx context.Context, jobID string) error {
	job := d.jobs[jobID]

	delete(d.jobs, jobID)
	delete(d.jobExecutions, jobID)
	delete(d.inProgress, jobID)
	delete(d.history, jobID)

	d.triggerEvent(jobstore.JobWatcher, jobstore.DeleteEvent, job)
	return nil
}

// helper method to read a single job from memory. This is used by both GetJob and GetJobs.
// It is important that we don't attempt to acquire a lock inside this method to avoid deadlocks since
// the callers are expected to be holding a lock, and golang doesn't support reentrant locks.
func (d *InMemoryJobStore) getJob(id string) (models.Job, error) {
	id, err := d.reifyJobID(id)
	if err != nil {
		return models.Job{}, err
	}

	j, ok := d.jobs[id]
	if !ok {
		returnError := bacerrors.NewJobNotFound(id)
		return models.Job{}, returnError
	}

	return j, nil
}

// reifyJobID ensures the provided job ID is a full-length ID. This is either through
// returning the ID, or resolving the short ID to a single job id.
func (d *InMemoryJobStore) reifyJobID(id string) (string, error) {
	if idgen.ShortID(id) == id {
		found := make([]string, 0, 1)
		// passed in a short id, need to resolve the long id first
		for k := range d.jobs {
			if idgen.ShortID(k) == id {
				found = append(found, k)
			}
		}
		switch len(found) {
		case 0:
			return "", bacerrors.NewJobNotFound(id)
		case 1:
			return found[0], nil
		default:
			return "", bacerrors.NewMultipleJobsFound(id, found)
		}
	}
	return id, nil
}

func (d *InMemoryJobStore) UpdateJobState(_ context.Context, request jobstore.UpdateJobStateRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// get the existing job state
	job, ok := d.jobs[request.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(request.JobID)
	}

	// check the expected state
	if err := request.Condition.Validate(job); err != nil {
		return err
	}
	if job.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(request.JobID, job.State.StateType, request.NewState)
	}

	// update the job state
	previousState := job.State.StateType
	job.State.StateType = request.NewState
	job.Revision++
	job.ModifyTime = d.clock.Now().UTC().UnixNano()
	d.jobs[request.JobID] = job
	if job.IsTerminal() {
		delete(d.inProgress, request.JobID)
	}
	d.appendJobHistory(job, previousState, request.Comment)

	d.triggerEvent(jobstore.JobWatcher, jobstore.UpdateEvent, job)

	return nil
}

func (d *InMemoryJobStore) CreateExecution(_ context.Context, execution models.Execution) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if _, ok := d.jobs[execution.JobID]; !ok {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}
	if _, ok := d.executions[execution.ID]; ok {
		return jobstore.NewErrExecutionAlreadyExists(execution.ID)
	}
	if execution.CreateTime == 0 {
		execution.CreateTime = d.clock.Now().UTC().UnixNano()
	}
	if execution.ModifyTime == 0 {
		execution.ModifyTime = execution.CreateTime
	}
	if execution.Revision == 0 {
		execution.Revision = 1
	}
	execution.Normalize()
	d.executions[execution.ID] = execution
	d.jobExecutions[execution.JobID] = append(d.jobExecutions[execution.JobID], execution.ID)
	d.appendExecutionHistory(execution, models.ExecutionStateNew, "")

	d.triggerEvent(jobstore.ExecutionWatcher, jobstore.CreateEvent, execution)

	return nil
}

func (d *InMemoryJobStore) UpdateExecution(_ context.Context, request jobstore.UpdateExecutionRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// find the existing execution
	existingExecution, ok := d.executions[request.ExecutionID]
	if !ok {
		return jobstore.NewErrExecutionNotFound(request.ExecutionID)
	}

	// check the expected state
	if err := request.Condition.Validate(existingExecution); err != nil {
		return err
	}
	if existingExecution.IsTerminalComputeState() {
		return jobstore.NewErrExecutionAlreadyTerminal(
			request.ExecutionID, existingExecution.ComputeState.StateType, request.NewValues.ComputeState.StateType)
	}

	// populate default values, maintain existing execution createTime
	newExecution := request.NewValues
	newExecution.CreateTime = existingExecution.CreateTime
	if newExecution.ModifyTime == 0 {
		newExecution.ModifyTime = d.clock.Now().UTC().UnixNano()
	}
	if newExecution.Revision == 0 {
		newExecution.Revision = existingExecution.Revision + 1
	}
	newExecution.Normalize()

	err := mergo.Merge(&newExecution, existingExecution)
	if err != nil {
		return err
	}

	// update the execution
	previousState := existingExecution.ComputeState.StateType
	d.executions[existingExecution.ID] = newExecution
	d.appendExecutionHistory(newExecution, previousState, request.Comment)

	d.triggerEvent(jobstore.ExecutionWatcher, jobstore.UpdateEvent, newExecution)

	return nil
}

// CreateEvaluation creates a new evaluation
func (d *InMemoryJobStore) CreateEvaluation(ctx context.Context, eval models.Evaluation) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	eval.Revision = 1

	_, ok := d.jobs[eval.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(eval.JobID)
	}

	_, ok = d.evaluations[eval.ID]
	if ok {
		return bacerrors.NewAlreadyExists(eval.ID, "Evaluation")
	}

	d.evaluations[eval.ID] = eval

	d.triggerEvent(jobstore.EvaluationWatcher, jobstore.CreateEvent, eval)

	return nil
}

// GetEvaluation retrieves the specified evaluation
func (d *InMemoryJobStore) GetEvaluation(ctx context.Context, id string) (models.Evaluation, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	ev, ok := d.evaluations[id]
	if !ok {
		returnError := bacerrors.NewEvaluationNotFound(id)
		return models.Evaluation{}, returnError
	}

	return ev, nil
}

func (d *InMemoryJobStore) GetEvaluationsByState(ctx context.Context, state string) ([]models.Evaluation, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	results := make([]models.Evaluation, 0, len(d.evaluations))

	for _, v := range d.evaluations {
		if v.Status == state {
			results = append(results, v)
		}
	}

	return results, nil
}

// UpdateEvaluation updates the stored evaluation
func (d *InMemoryJobStore) UpdateEvaluation(ctx context.Context, update jobstore.UpdateEvaluationRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	eval := d.evaluations[update.EvaluationID]

	if err := update.Condition.Validate(eval); err != nil {
		return err
	}

	eval.Status = update.NewStatus
	eval.ModifyTime = update.ModificationTime
	eval.Revision = update.Revision

	d.evaluations[update.EvaluationID] = eval

	d.triggerEvent(jobstore.EvaluationWatcher, jobstore.UpdateEvent, eval)
	return nil
}

// DeleteEvaluation deletes the specified evaluation
func (d *InMemoryJobStore) DeleteEvaluation(ctx context.Context, id string) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.triggerEvent(jobstore.EvaluationWatcher, jobstore.DeleteEvent, d.evaluations[id])

	delete(d.evaluations, id)
	return nil
}

func (d *InMemoryJobStore) Close(_ context.Context) error {
	return nil
}

func (d *InMemoryJobStore) appendJobHistory(updateJob models.Job, previousState models.JobStateType, comment string) {
	historyEntry := models.JobHistory{
		Type:  models.JobHistoryTypeJobLevel,
		JobID: updateJob.ID,
		JobState: &models.StateChange[models.JobStateType]{
			Previous: previousState,
			New:      updateJob.State.StateType,
		},
		NewRevision: updateJob.Revision,
		Comment:     comment,
		Time:        time.Unix(0, updateJob.ModifyTime),
	}
	d.history[updateJob.ID] = append(d.history[updateJob.ID], historyEntry)
}

func (d *InMemoryJobStore) appendExecutionHistory(updatedExecution models.Execution,
	previousState models.ExecutionStateType, comment string) {
	historyEntry := models.JobHistory{
		Type:        models.JobHistoryTypeExecutionLevel,
		JobID:       updatedExecution.JobID,
		NodeID:      updatedExecution.NodeID,
		ExecutionID: updatedExecution.ID,
		ExecutionState: &models.StateChange[models.ExecutionStateType]{
			Previous: previousState,
			New:      updatedExecution.ComputeState.StateType,
		},
		NewRevision: updatedExecution.Revision,
		Comment:     comment,
		Time:        time.Unix(0, updatedExecution.ModifyTime),
	}
	d.history[updatedExecution.JobID] = append(d.history[updatedExecution.JobID], historyEntry)
}

// Static check to ensure that Transport implements Transport:
var _ jobstore.Store = (*InMemoryJobStore)(nil)
