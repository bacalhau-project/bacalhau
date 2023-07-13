package boltjobstore

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/benbjohnson/clock"
	"github.com/imdario/mergo"
	"github.com/samber/lo"
	bolt "go.etcd.io/bbolt"
)

const (
	BucketPathJobs             = "jobs"
	BucketPathTags             = "jobs.tags"
	BucketPathState            = "jobs.state"
	BucketPathJobHistory       = "jobs.history"
	BucketPathInProgress       = "jobs.inprogress"
	BucketPathClients          = "jobs.clients"
	BucketPathExecutions       = "executions"
	BucketPathExecutionHistory = "executions.history"

	newJobComment = "Job created"
)

var BucketJobs = []byte("jobs")
var BucketJobsTags = []byte("tags")
var BucketJobsState = []byte("state")
var BucketJobsInProgress = []byte("inprogress")
var BucketJobsHistory = []byte("history")
var BucketJobsClients = []byte("clients")
var BucketExecutions = []byte("executions")

type BoltJobStore struct {
	database *bolt.DB
	clock    clock.Clock

	inProgressIndex *Index
	clientsIndex    *Index
	tagsIndex       *Index
}

type Option func(store *BoltJobStore)

func WithClock(clock clock.Clock) Option {
	return func(store *BoltJobStore) {
		store.clock = clock
	}
}

// NewBoltJobStore creates is a boltdb-backed JobStore implementation, storing
// information about jobs and their state in a structure that allows for fast
// lookup by ID, and slightly slower lookup by other criteria that are encoded
// in buckets.
//
// * In progress jobs are kept in an Index within the inprogress bucket,
// within the job bucket.
//
//	jobs
//	 |---> inprogress
//	           |----> JOBID
//
// * Job state are stored in a jobs sub-bucket called state and this maps the
// job id against the current state of the job.
//
//	jobs
//	 |---> state
//	           |----> key:JobID -> value:jobstate
//
// * Job history entities are stored in a history sub-bucket that itself
// constains a bucket labeled with the job id.  Inside this bucket, each
// key is a three digit sequence number to provide ordering for the retrieval.
//
//		jobs
//		 |---> history
//		           |----> JobID
//	                     |----> key:sequence, value:history
//
// * Within the jobs bucket, the clients bucket is an index where the label
// is the client ID, and the identified bucket is the job ID.
//
//	jobs
//	  |---- clients # Contains marker keys for client jobs
//		          |---- <client-id> # A specific client ID
//		                       |---- JOBID
//
// * Tags are stored in a tags index bucket that is within the top level
// jobs bucket. Each bucket within the tags bucket is itself a tag, and
// contains a list of keys (also bucket).
//
//	   jobs
//		|---- tags # Tags used in jobs for inclusion/exclusion search
//		        |---- <tag> # A specific tag name
//		                |---- JOBID
//
// * The actual job data is available within the jobs bucket directly
// where the key is the job id and the value the JSON encoded object.
//
//	   jobs
//		|--- key:JobID -> value: {JobObject}
//
// * Executions are also stored in the job store, with a top level
// executions bucket containing a bucket for each execution-id. Within
// that bucket a key of 'data' has a value that contains the execution
// state, and a 'history' bucket contains a sequence of keys pointing
// to an ExecutionHistory - each of these sequential keys is a logical
// counter and guaranteed to be in sequence allowing for a lexicographic
// retrieval.
//
//		executions
//			|--- <execution-id> # For each execution
//			      |--- key:data value:{ExecutionState}
//			|--- history # execution history
//	              |---  <job-id>
//			                |--- key:nnn -> value:{ExecutionHistory}
func NewBoltJobStore(dbPath string, options ...Option) (*BoltJobStore, error) {
	db, err := GetDatabase(dbPath)
	if err != nil {
		return nil, err
	}

	store := &BoltJobStore{
		database: db,
		clock:    clock.New(),
	}

	for _, opt := range options {
		opt(store)
	}

	// Create the top level buckets ready for use as they
	// will definitely be required
	err = db.Update(func(tx *bolt.Tx) (err error) {
		var root *bolt.Bucket

		root, err = tx.CreateBucketIfNotExists(BucketJobs)
		if err != nil {
			return err
		}

		// Create the buckets underneath the top level jobs bucket
		subBuckets := [][]byte{
			BucketJobsTags,
			BucketJobsInProgress,
			BucketJobsClients,
			BucketJobsState,
			BucketJobsHistory,
		}

		for _, sub := range subBuckets {
			_, err = root.CreateBucketIfNotExists(sub)
			if err != nil {
				return err
			}
		}

		if exec, err := tx.CreateBucketIfNotExists(BucketExecutions); err != nil {
			return err
		} else {
			if _, err = exec.CreateBucketIfNotExists([]byte("history")); err != nil {
				return err
			}
		}

		return nil
	})

	store.inProgressIndex = NewIndex(BucketPathJobs)
	store.clientsIndex = NewIndex(BucketPathClients)
	store.tagsIndex = NewIndex(BucketPathTags)

	return store, err
}

// GetJob retrieves the Job identified by the id string. If the job isn't found it will
// return an indicating the error.
func (b *BoltJobStore) GetJob(ctx context.Context, id string) (model.Job, error) {
	var job model.Job
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		job, err = b.getJob(tx, id)
		return
	})
	return job, err
}

func (b *BoltJobStore) getJob(tx *bolt.Tx, id string) (model.Job, error) {
	var job model.Job

	data := GetBucketData(tx, string(BucketJobs), []byte(id))
	if data == nil {
		return job, bacerrors.NewJobNotFound(id)
	}

	err := json.Unmarshal(data, &job)
	return job, err
}

func (b *BoltJobStore) getExecution(tx *bolt.Tx, executionID model.ExecutionID) (model.ExecutionState, error) {
	var exec model.ExecutionState

	bucket, err := NewBucketPath(BucketPathExecutions).Get(tx, false)
	if err != nil {
		return exec, err
	}

	data := bucket.Get([]byte(executionID.String()))
	if data == nil {
		return exec, jobstore.NewErrExecutionNotFound(executionID)
	}

	err = json.Unmarshal(data, &exec)
	return exec, err
}

func (b *BoltJobStore) jobExists(tx *bolt.Tx, jobID string) bool {
	_, err := b.getJob(tx, jobID)
	return err == nil
}

// GetJobs returns all Jobs that match the provided query
func (b *BoltJobStore) GetJobs(ctx context.Context, query jobstore.JobQuery) ([]model.Job, error) {
	var jobs []model.Job
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		jobs, err = b.getJobs(tx, query)
		return
	})
	return jobs, err
}

func (b *BoltJobStore) getJobs(tx *bolt.Tx, query jobstore.JobQuery) ([]model.Job, error) {
	if query.ID != "" {
		job, err := b.getJob(tx, query.ID)
		return []model.Job{job}, err
	}

	jobSet := make(map[string]struct{})

	if query.ReturnAll {
		bkt, err := NewBucketPath(BucketPathJobs).Get(tx, false)
		if err != nil {
			return nil, err
		}

		err = bkt.ForEach(func(k []byte, v []byte) error {
			if v != nil { // If not a bucket
				jobSet[string(k)] = struct{}{}
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		if query.ClientID != "" {
			ids, err := b.clientsIndex.List(tx, []byte(query.ClientID))
			if err != nil {
				return nil, err
			}

			for _, k := range ids {
				jobSet[string(k)] = struct{}{}
			}
		}

		for _, tag := range query.IncludeTags {
			tagLabel := []byte(strings.ToLower(string(tag)))
			ids, err := b.tagsIndex.List(tx, tagLabel)
			if err != nil {
				return nil, err
			}

			for _, k := range ids {
				jobSet[string(k)] = struct{}{}
			}
		}
	}

	for _, tag := range query.ExcludeTags {
		tagLabel := []byte(strings.ToLower(string(tag)))
		ids, err := b.tagsIndex.List(tx, tagLabel)
		if err != nil {
			return nil, err
		}

		for _, k := range ids {
			delete(jobSet, string(k))
		}
	}

	var result []model.Job

	bucket := tx.Bucket(BucketJobs)
	for key := range jobSet {
		var job model.Job
		data := bucket.Get([]byte(key))
		err := json.Unmarshal(data, &job)
		if err != nil {
			return nil, err
		}
		result = append(result, job)
	}

	listSorter := b.getListSorter(result, query)
	sort.Slice(result, listSorter)

	limit := query.Limit
	if limit == 0 {
		limit = len(result)
	} else {
		limit = math.Min(len(result), limit+query.Offset)
	}

	return result[query.Offset:limit], nil
}

func (b *BoltJobStore) getListSorter(jobs []model.Job, query jobstore.JobQuery) func(i, j int) bool {
	return func(i, j int) bool {
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
}

// GetJobState returns the current job state for the provided job id
func (b *BoltJobStore) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	var state model.JobState
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		state, err = b.getJobState(tx, jobID)
		return
	})

	return state, err
}

func (b *BoltJobStore) getJobState(tx *bolt.Tx, jobID string) (model.JobState, error) {
	var state model.JobState

	data := GetBucketData(tx, string(BucketJobsState), []byte(jobID))
	if data == nil {
		return state, bacerrors.NewJobNotFound(jobID)
	}

	err := json.Unmarshal(data, &state)
	return state, err
}

// GetInProgressJobs gets a list of the currently in-progress jobs
func (b *BoltJobStore) GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error) {
	var infos []model.JobWithInfo
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		infos, err = b.getInProgressJobs(tx)
		return
	})
	return infos, err
}

func (b *BoltJobStore) getInProgressJobs(tx *bolt.Tx) ([]model.JobWithInfo, error) {
	var infos []model.JobWithInfo
	var keys [][]byte

	keys, err := b.inProgressIndex.List(tx, BucketJobsInProgress)
	if err != nil {
		return nil, err
	}

	bktJobs := tx.Bucket(BucketJobs)
	bktState, err := NewBucketPath(BucketPathState).Get(tx, false)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		var job model.Job
		var jobState model.JobState

		dataJob := bktJobs.Get(key)
		err = json.Unmarshal(dataJob, &job)
		if err != nil {
			return nil, err
		}

		dataState := bktState.Get(key)
		err = json.Unmarshal(dataState, &jobState)
		if err != nil {
			return nil, err
		}

		info := model.JobWithInfo{
			Job:   job,
			State: jobState,
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// GetJobHistory returns the job (and execution) history for the provided options
func (b *BoltJobStore) GetJobHistory(ctx context.Context,
	jobID string,
	options jobstore.JobHistoryFilterOptions) ([]model.JobHistory, error) {
	var history []model.JobHistory
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		history, err = b.getJobHistory(tx, jobID, options)
		return
	})

	return history, err
}

func (b *BoltJobStore) getJobHistory(tx *bolt.Tx, jobID string,
	options jobstore.JobHistoryFilterOptions) ([]model.JobHistory, error) {
	var history []model.JobHistory

	if !options.ExcludeJobLevel {
		if bkt, err := NewBucketPath(BucketPathJobHistory, jobID).Get(tx, false); err != nil {
			return nil, err
		} else {
			err = bkt.ForEach(func(key []byte, data []byte) error {
				var item model.JobHistory

				err := json.Unmarshal(data, &item)
				if err != nil {
					return err
				}

				history = append(history, item)
				return nil
			})

			if err != nil {
				return nil, err
			}
		}
	}

	if !options.ExcludeExecutionLevel {
		// 	// Get the executions for this JobID
		if bkt, err := NewBucketPath(BucketPathExecutionHistory, jobID).Get(tx, false); err != nil {
			return nil, err
		} else {
			err = bkt.ForEach(func(key []byte, data []byte) error {
				var item model.JobHistory

				err := json.Unmarshal(data, &item)
				if err != nil {
					return err
				}

				history = append(history, item)
				return nil
			})

			if err != nil {
				return nil, err
			}
		}
	}

	// Filter out anything before the specified Since time
	history = lo.Filter(history, func(item model.JobHistory, index int) bool {
		return item.Time.Unix() >= options.Since
	})

	sort.Slice(history, func(i, j int) bool { return history[i].Time.UTC().Before(history[j].Time.UTC()) })

	return history, nil
}

// CreateJob creates a new record of a job in the data store
func (b *BoltJobStore) CreateJob(ctx context.Context, job model.Job) error {
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.createJob(tx, job)
	})
}

func (b *BoltJobStore) createJob(tx *bolt.Tx, job model.Job) error {
	if b.jobExists(tx, job.ID()) {
		return jobstore.NewErrJobAlreadyExists(job.Metadata.ID)
	}

	jobIDKey := []byte(job.Metadata.ID)

	jobState := model.JobState{
		JobID:      job.Metadata.ID,
		State:      model.JobStateNew,
		Version:    1,
		CreateTime: b.clock.Now().UTC(),
		UpdateTime: b.clock.Now().UTC(),
	}
	data, err := json.Marshal(jobState)
	if err != nil {
		return err
	}

	// Write the JobState to the state bucket
	if bkt, err := NewBucketPath(BucketPathState).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put(jobIDKey, data); err != nil {
			return err
		}
	}

	// Write the job to the Job bucket
	data, err = json.Marshal(job)
	if err != nil {
		return err
	}
	if bkt, err := NewBucketPath(BucketPathJobs).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put(jobIDKey, data); err != nil {
			return err
		}
	}

	if err = b.inProgressIndex.Add(tx, BucketJobsInProgress, jobIDKey); err != nil {
		return err
	}

	if err = b.clientsIndex.Add(tx, []byte(job.Metadata.ClientID), jobIDKey); err != nil {
		return err
	}

	// Write sentinels keys for specific tags
	for _, tag := range job.Spec.Annotations {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Add(tx, tagBytes, jobIDKey); err != nil {
			return err
		}
	}

	return b.appendJobHistory(tx, jobState, model.JobStateNew, newJobComment)
}

// DeleteJob removes the specified job from the system entirely
func (b *BoltJobStore) DeleteJob(ctx context.Context, jobID string) error {
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.deleteJob(tx, jobID)
	})
}

func (b *BoltJobStore) deleteJob(tx *bolt.Tx, jobID string) error {
	jobIDKey := []byte(jobID)

	job, err := b.getJob(tx, jobID)
	if err != nil {
		return bacerrors.NewJobNotFound(jobID)
	}

	// Delete the JobState from the state bucket
	if bkt, err := NewBucketPath(BucketPathState).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Delete(jobIDKey); err != nil {
			return err
		}
	}

	// Delete the actual job
	if bkt, err := NewBucketPath(BucketPathJobs).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Delete(jobIDKey); err != nil {
			return err
		}
	}

	if err = b.inProgressIndex.Remove(tx, BucketJobsInProgress, jobIDKey); err != nil {
		return err
	}

	if err = b.clientsIndex.Remove(tx, []byte(job.Metadata.ClientID), jobIDKey); err != nil {
		return err
	}

	// Delete sentinels keys for specific tags
	for _, tag := range job.Spec.Annotations {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Remove(tx, tagBytes, jobIDKey); err != nil {
			return err
		}
	}

	return nil
}

// UpdateJobState updates the current state for a single Job, appending an entry to
// the history at the same time
func (b *BoltJobStore) UpdateJobState(ctx context.Context, request jobstore.UpdateJobStateRequest) error {
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.updateJobState(tx, request)
	})
}

func (b *BoltJobStore) updateJobState(tx *bolt.Tx, request jobstore.UpdateJobStateRequest) error {
	job, err := b.getJob(tx, request.JobID)
	if err != nil {
		return jobstore.NewErrJobNotFound(request.JobID)
	}

	bucket, err := NewBucketPath(BucketPathState).Get(tx, false)
	if err != nil {
		return err
	}

	data := bucket.Get([]byte(job.ID()))
	if data == nil {
		return jobstore.NewErrJobNotFound(job.ID())
	}

	var jobState model.JobState
	err = json.Unmarshal(data, &jobState)
	if err != nil {
		return err
	}

	// check the expected state
	if err := request.Condition.Validate(jobState); err != nil {
		return err
	}

	if jobState.State.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(job.ID(), jobState.State, request.NewState)
	}

	// update the job state
	previousState := jobState.State
	jobState.State = request.NewState
	jobState.Version++
	jobState.UpdateTime = b.clock.Now().UTC()

	data, err = json.Marshal(jobState)
	if err != nil {
		return err
	}

	err = bucket.Put([]byte(request.JobID), data)
	if err != nil {
		return err
	}

	if request.NewState.IsTerminal() {
		if bkt, err := NewBucketPath(BucketPathInProgress).Get(tx, false); err == nil {
			_ = bkt.Delete([]byte(request.JobID))
		}
	}

	return b.appendJobHistory(tx, jobState, previousState, request.Comment)
}

func (b *BoltJobStore) appendJobHistory(tx *bolt.Tx, updateJob model.JobState, previousState model.JobStateType, comment string) error {
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
	data, err := json.Marshal(historyEntry)
	if err != nil {
		return err
	}

	// Get the history bucket for this job ID, which involves potentially
	// creating the bucket (jobs.history.JOBID)
	if bkt, err := NewBucketPath(BucketPathJobHistory, updateJob.JobID).Get(tx, true); err != nil {
		return err
	} else {
		seq := BucketSequenceString(tx, bkt)
		if err = bkt.Put([]byte(seq), data); err != nil {
			return err
		}
	}

	return nil
}

// CreateExecution creates a record of a new execution
func (b *BoltJobStore) CreateExecution(ctx context.Context, execution model.ExecutionState) error {
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.createExecution(tx, execution)
	})
}

func (b *BoltJobStore) createExecution(tx *bolt.Tx, execution model.ExecutionState) error {
	if !b.jobExists(tx, execution.JobID) {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}

	execID := []byte(execution.ID().String())

	if execution.CreateTime.IsZero() {
		execution.CreateTime = b.clock.Now().UTC()
	}
	if execution.UpdateTime.IsZero() {
		execution.UpdateTime = execution.CreateTime
	}
	if execution.Version == 0 {
		execution.Version = 1
	}

	// Check for the existence of this ID and if it doesn't already exist, then create
	// it
	if bucket, err := NewBucketPath(BucketPathExecutions).Get(tx, false); err != nil {
		return err
	} else {
		_, err := b.getExecution(tx, execution.ID())
		if err == nil {
			return jobstore.NewErrExecutionAlreadyExists(execution.ID())
		}

		if data, err := json.Marshal(execution); err != nil {
			return err
		} else {
			err = bucket.Put(execID, data)
			if err != nil {
				return err
			}
		}
	}

	return b.appendExecutionHistory(tx, execution, model.ExecutionStateNew, "")
}

// UpdateExecution updates the state of a single execution by loading from storage,
// updating and then writing back in a single transaction
func (b *BoltJobStore) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) error {
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.updateExecution(tx, request)
	})
}

func (b *BoltJobStore) updateExecution(tx *bolt.Tx, request jobstore.UpdateExecutionRequest) error {
	if !b.jobExists(tx, request.ExecutionID.JobID) {
		return jobstore.NewErrJobNotFound(request.ExecutionID.JobID)
	}

	existingExecution, err := b.getExecution(tx, request.ExecutionID)
	if err != nil {
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
		newExecution.CreateTime = b.clock.Now().UTC()
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

	data, err := json.Marshal(newExecution)
	if err != nil {
		return err
	}

	bucket, err := NewBucketPath(BucketPathExecutions).Get(tx, false)
	if err != nil {
		return err
	} else {
		err = bucket.Put([]byte(newExecution.ID().String()), data)
		if err != nil {
			return err
		}
	}

	return b.appendExecutionHistory(tx, newExecution, existingExecution.State, request.Comment)
}

func (b *BoltJobStore) appendExecutionHistory(tx *bolt.Tx, updated model.ExecutionState,
	previous model.ExecutionStateType, cmt string) error {
	historyEntry := model.JobHistory{
		Type:             model.JobHistoryTypeExecutionLevel,
		JobID:            updated.JobID,
		NodeID:           updated.NodeID,
		ComputeReference: updated.ComputeReference,
		ExecutionState: &model.StateChange[model.ExecutionStateType]{
			Previous: previous,
			New:      updated.State,
		},
		NewVersion: updated.Version,
		Comment:    cmt,
		Time:       updated.UpdateTime,
	}

	data, err := json.Marshal(historyEntry)
	if err != nil {
		return err
	}

	// Get the history bucket for this job ID, which involves potentially
	// creating the bucket (executions.history.<jobid>)
	if bkt, err := NewBucketPath(BucketPathExecutionHistory, updated.JobID).Get(tx, true); err != nil {
		return err
	} else {
		seq := BucketSequenceString(tx, bkt)
		if err = bkt.Put([]byte(seq), data); err != nil {
			return err
		}
	}

	return nil
}

func (b *BoltJobStore) Close(ctx context.Context) error {
	return b.database.Close()
}
