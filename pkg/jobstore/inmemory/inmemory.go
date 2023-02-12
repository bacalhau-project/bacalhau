package inmemory

import (
	"context"
	"sort"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/imdario/mergo"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/jobstore"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

const newJobComment = "Job created"

type JobStore struct {
	// we keep pointers to these things because we will update them partially
	jobs       map[string]model.Job
	states     map[string]model.JobState
	history    map[string][]model.JobHistory
	inprogress map[string]struct{}
	mtx        sync.RWMutex
}

func NewJobStore() *JobStore {
	res := &JobStore{
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
func (d *JobStore) GetJob(_ context.Context, id string) (model.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getJob(id)
}

func (d *JobStore) GetJobs(ctx context.Context, query jobstore.JobQuery) ([]model.Job, error) {
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

func (d *JobStore) GetJobState(_ context.Context, jobID string) (model.JobState, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	state, ok := d.states[jobID]
	if !ok {
		return model.JobState{}, bacerrors.NewJobNotFound(jobID)
	}
	return state, nil
}

func (d *JobStore) GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error) {
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

func (d *JobStore) GetJobHistory(_ context.Context, jobID string) ([]model.JobHistory, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	history, ok := d.history[jobID]
	if !ok {
		return nil, jobstore.NewErrJobNotFound(jobID)
	}
	return history, nil
}

func (d *JobStore) GetJobsCount(ctx context.Context, query jobstore.JobQuery) (int, error) {
	useQuery := query
	useQuery.Limit = 0
	useQuery.Offset = 0
	jobs, err := d.GetJobs(ctx, useQuery)
	if err != nil {
		return 0, err
	}
	return len(jobs), nil
}

func (d *JobStore) CreateJob(_ context.Context, job model.Job) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	existingJob, ok := d.jobs[job.Metadata.ID]
	if ok {
		return jobstore.NewErrJobAlreadyExists(existingJob.Metadata.ID)
	}
	d.jobs[job.Metadata.ID] = job

	// populate shard states
	shardStates := make(map[int]model.ShardState, job.Spec.ExecutionPlan.TotalShards)
	for i := 0; i < job.Spec.ExecutionPlan.TotalShards; i++ {
		shardStates[i] = model.ShardState{
			JobID:      job.Metadata.ID,
			ShardIndex: i,
			State:      model.ShardStateInProgress,
			Version:    1,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
	}

	// populate job state
	jobState := model.JobState{
		JobID:      job.Metadata.ID,
		Shards:     shardStates,
		State:      model.JobStateInProgress,
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
func (d *JobStore) getJob(id string) (model.Job, error) {
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

func (d *JobStore) UpdateJobState(_ context.Context, request jobstore.UpdateJobStateRequest) error {
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

func (d *JobStore) GetShardState(_ context.Context, shardID model.ShardID) (model.ShardState, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	jobState, ok := d.states[shardID.JobID]
	if !ok {
		return model.ShardState{}, jobstore.NewErrJobNotFound(shardID.JobID)
	}
	shardState, ok := jobState.Shards[shardID.Index]
	if !ok {
		return model.ShardState{}, jobstore.NewErrShardNotFound(shardID)
	}
	return shardState, nil
}

func (d *JobStore) UpdateShardState(_ context.Context, request jobstore.UpdateShardStateRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// find the existing shard
	jobState, ok := d.states[request.ShardID.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(request.ShardID.JobID)
	}
	shardState, ok := jobState.Shards[request.ShardID.Index]
	if !ok {
		return jobstore.NewErrShardNotFound(request.ShardID)
	}

	// check the expected state
	if err := request.Condition.Validate(shardState); err != nil {
		return err
	}
	if shardState.State.IsTerminal() {
		return jobstore.NewErrShardAlreadyTerminal(request.ShardID, shardState.State, request.NewState)
	}

	// update the shard state
	previousState := shardState.State
	shardState.State = request.NewState
	shardState.Version++
	shardState.UpdateTime = time.Now()
	jobState.Shards[request.ShardID.Index] = shardState
	d.states[request.ShardID.JobID] = jobState
	d.appendShardHistory(shardState, previousState, request.Comment)
	return nil
}

func (d *JobStore) CreateExecution(_ context.Context, execution model.ExecutionState) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	jobState, ok := d.states[execution.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}
	shardState, ok := jobState.Shards[execution.ShardIndex]
	if !ok {
		return jobstore.NewErrShardNotFound(execution.ShardID())
	}
	for _, e := range shardState.Executions {
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
	shardState.Executions = append(shardState.Executions, execution)
	jobState.Shards[execution.ShardIndex] = shardState
	d.states[execution.JobID] = jobState
	d.appendExecutionHistory(execution, model.ExecutionStateNew, "")
	return nil
}

func (d *JobStore) UpdateExecution(_ context.Context, request jobstore.UpdateExecutionRequest) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// find the existing execution
	jobState, ok := d.states[request.ExecutionID.JobID]
	if !ok {
		return jobstore.NewErrJobNotFound(request.ExecutionID.JobID)
	}
	shardState, ok := jobState.Shards[request.ExecutionID.ShardIndex]
	if !ok {
		return jobstore.NewErrShardNotFound(request.ExecutionID.ShardID())
	}
	var existingExecution model.ExecutionState
	executionIndex := -1
	for i, e := range shardState.Executions {
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
	shardState.Executions[executionIndex] = newExecution
	jobState.Shards[newExecution.ShardIndex] = shardState
	d.states[newExecution.JobID] = jobState
	d.appendExecutionHistory(newExecution, previousState, request.Comment)
	return nil
}

func (d *JobStore) appendJobHistory(updateJob model.JobState, previousState model.JobStateType, comment string) {
	historyEntry := model.JobHistory{
		Type:          model.JobHistoryTypeJobLevel,
		JobID:         updateJob.JobID,
		PreviousState: previousState.String(),
		NewState:      updateJob.State.String(),
		NewVersion:    updateJob.Version,
		Comment:       comment,
		Time:          updateJob.UpdateTime,
	}
	d.history[updateJob.JobID] = append(d.history[updateJob.JobID], historyEntry)
}

func (d *JobStore) appendShardHistory(updatedShard model.ShardState, previousState model.ShardStateType, comment string) {
	historyEntry := model.JobHistory{
		Type:          model.JobHistoryTypeShardLevel,
		JobID:         updatedShard.JobID,
		ShardIndex:    updatedShard.ShardIndex,
		PreviousState: previousState.String(),
		NewState:      updatedShard.State.String(),
		NewVersion:    updatedShard.Version,
		Comment:       comment,
		Time:          updatedShard.UpdateTime,
	}
	d.history[updatedShard.JobID] = append(d.history[updatedShard.JobID], historyEntry)
}

func (d *JobStore) appendExecutionHistory(updatedExecution model.ExecutionState, previousState model.ExecutionStateType, comment string) {
	historyEntry := model.JobHistory{
		Type:             model.JobHistoryTypeExecutionLevel,
		JobID:            updatedExecution.JobID,
		ShardIndex:       updatedExecution.ShardIndex,
		NodeID:           updatedExecution.NodeID,
		ComputeReference: updatedExecution.ComputeReference,
		PreviousState:    previousState.String(),
		NewState:         updatedExecution.State.String(),
		NewStateType:     updatedExecution.State,
		NewVersion:       updatedExecution.Version,
		Comment:          comment,
		Time:             updatedExecution.UpdateTime,
	}
	d.history[updatedExecution.JobID] = append(d.history[updatedExecution.JobID], historyEntry)
}

// Static check to ensure that Transport implements Transport:
var _ jobstore.Store = (*JobStore)(nil)
