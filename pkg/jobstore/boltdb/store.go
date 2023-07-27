package boltjobstore

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/benbjohnson/clock"
	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/exp/maps"
)

const (
	BucketJobs              = "jobs"
	BucketJobsTags          = "jobs_tags"
	BucketJobsState         = "jobs_state"
	BucketJobsInProgress    = "jobs_inprogress"
	BucketJobsHistory       = "jobs_history"
	BucketJobsClients       = "jobs_clients"
	BucketExecutions        = "executions"
	BucketExecutionsHistory = "executions_history"
	BucketEvaluations       = "evaluations"

	newJobComment = "Job created"
)

var BucketJobsBytes = []byte(BucketJobs)
var BucketJobsTagsBytes = []byte(BucketJobsTags)
var BucketJobsStateBytes = []byte(BucketJobsState)
var BucketJobsInProgressBytes = []byte(BucketJobsInProgress)
var BucketJobsHistoryBytes = []byte(BucketJobsHistory)
var BucketJobsClientsBytes = []byte(BucketJobsClients)
var BucketExecutionsBytes = []byte(BucketExecutions)
var BucketExecutionsHistoryBytes = []byte(BucketExecutionsHistory)
var BucketEvaluationsBytes = []byte(BucketEvaluations)

type BoltJobStore struct {
	database    *bolt.DB
	clock       clock.Clock
	watchers    []*jobstore.Watcher
	watcherLock sync.Mutex

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
// * Evaluations are kept within the top level evaluations bucket and
// referenced by their ID
//
//	evaluations
//	     |---- key:EvaluationID -> value: {Evaluation}
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
		watchers: make([]*jobstore.Watcher, 0), //nolint:gomnd
	}

	for _, opt := range options {
		opt(store)
	}

	// Create the top level buckets ready for use as they
	// will definitely be required
	err = db.Update(func(tx *bolt.Tx) (err error) {
		buckets := [][]byte{
			BucketJobsBytes,
			BucketJobsTagsBytes,
			BucketJobsInProgressBytes,
			BucketJobsClientsBytes,
			BucketJobsStateBytes,
			BucketJobsHistoryBytes,
			BucketExecutionsBytes,
			BucketExecutionsHistoryBytes,
			BucketEvaluationsBytes,
		}

		for _, bkt := range buckets {
			_, err = tx.CreateBucketIfNotExists(bkt)
			if err != nil {
				return err
			}
		}

		return nil
	})

	log.Debug().Str("DBFile", dbPath).Msg("created bolt-backed job store")

	store.inProgressIndex = NewIndex(BucketJobsInProgress)
	store.clientsIndex = NewIndex(BucketJobsClients)
	store.tagsIndex = NewIndex(BucketJobsTags)

	return store, err
}

func (b *BoltJobStore) Watch(ctx context.Context,
	types jobstore.StoreWatcherType,
	events jobstore.StoreEventType) chan jobstore.WatchEvent {
	w := jobstore.NewWatcher(types, events)

	b.watcherLock.Lock() // keep the watchers lock as narrow as possible
	b.watchers = append(b.watchers, w)
	b.watcherLock.Unlock()

	return w.Channel()
}

func (b *BoltJobStore) triggerEvent(t jobstore.StoreWatcherType, e jobstore.StoreEventType, object interface{}) {
	data, _ := json.Marshal(object)

	for _, w := range b.watchers {
		if !w.IsWatchingEvent(e) || !w.IsWatchingType(t) {
			return
		}

		_ = w.WriteEvent(t, e, data, false) // Do not block
	}
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

	bucket, err := NewBucketPath(BucketExecutions).Get(tx, false)
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
	tagSet := make(map[string]struct{})
	clientSet := make(map[string]struct{})

	if query.ReturnAll {
		bkt, err := NewBucketPath(BucketJobs).Get(tx, false)
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
		for _, tag := range query.IncludeTags {
			tagLabel := []byte(strings.ToLower(string(tag)))
			ids, err := b.tagsIndex.List(tx, tagLabel)
			if err != nil {
				return nil, err
			}

			for _, k := range ids {
				tagSet[string(k)] = struct{}{}
			}
		}

		if query.ClientID != "" {
			ids, err := b.clientsIndex.List(tx, []byte(query.ClientID))
			if err != nil {
				return nil, err
			}

			for _, k := range ids {
				clientSet[string(k)] = struct{}{}
			}
		}

		clientKeys := maps.Keys(clientSet)
		clientKeysLen := len(clientKeys)

		tagKeys := maps.Keys(tagSet)
		tagKeysLen := len(tagKeys)

		var jobIDs []string
		if clientKeysLen > 0 && tagKeysLen > 0 {
			jobIDs = lo.Intersect(clientKeys, tagKeys)
		} else if clientKeysLen > 0 && tagKeysLen == 0 { // being explicit
			jobIDs = clientKeys
		} else if tagKeysLen > 0 && clientKeysLen == 0 { // being explicit
			jobIDs = tagKeys
		}

		lo.ForEach[string](jobIDs, func(item string, _ int) {
			jobSet[item] = struct{}{}
		})
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

	bucket, _ := NewBucketPath(BucketJobs).Get(tx, false)
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

	data := GetBucketData(tx, BucketJobsState, []byte(jobID))
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

	keys, err := b.inProgressIndex.List(tx)
	if err != nil {
		return nil, err
	}

	bktJobs, err := NewBucketPath(BucketJobs).Get(tx, false)
	if err != nil {
		return nil, err
	}

	bktState, err := NewBucketPath(BucketJobsState).Get(tx, false)
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
		if bkt, err := NewBucketPath(BucketJobsHistory, jobID).Get(tx, false); err != nil {
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
		if bkt, err := NewBucketPath(BucketExecutionsHistory, jobID).Get(tx, false); err != nil {
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

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.JobWatcher, jobstore.CreateEvent, job)
	})

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
	if bkt, err := NewBucketPath(BucketJobsState).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put(jobIDKey, data); err != nil {
			return err
		}
	}

	// Write the job to the Job bucket
	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if bkt, err := NewBucketPath(BucketJobs).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put(jobIDKey, jobData); err != nil {
			return err
		}
	}

	if err = b.inProgressIndex.Add(tx, jobIDKey); err != nil {
		return err
	}

	if job.Metadata.ClientID == "" {
		return errors.New("job is missing a client id")
	}

	if err = b.clientsIndex.Add(tx, jobIDKey, []byte(job.Metadata.ClientID)); err != nil {
		return err
	}

	// Write sentinels keys for specific tags
	for _, tag := range job.Spec.Annotations {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Add(tx, jobIDKey, tagBytes); err != nil {
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

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.JobWatcher, jobstore.DeleteEvent, job)
	})

	// Delete the JobState from the state bucket
	if bkt, err := NewBucketPath(BucketJobsState).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Delete(jobIDKey); err != nil {
			return err
		}
	}

	// Delete the actual job
	if bkt, err := NewBucketPath(BucketJobs).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Delete(jobIDKey); err != nil {
			return err
		}
	}

	if err = b.inProgressIndex.Remove(tx, jobIDKey); err != nil {
		return err
	}

	if err = b.clientsIndex.Remove(tx, jobIDKey, []byte(job.Metadata.ClientID)); err != nil {
		return err
	}

	// Delete sentinels keys for specific tags
	for _, tag := range job.Spec.Annotations {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Remove(tx, jobIDKey, tagBytes); err != nil {
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
	bucket, err := NewBucketPath(BucketJobsState).Get(tx, false)
	if err != nil {
		return err
	}

	jobIDBytes := []byte(request.JobID)

	data := bucket.Get(jobIDBytes)
	if data == nil {
		return jobstore.NewErrJobNotFound(request.JobID)
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
		return jobstore.NewErrJobAlreadyTerminal(request.JobID, jobState.State, request.NewState)
	}

	// Setup an oncommit handler after the obvious errors/checks
	tx.OnCommit(func() {
		b.triggerEvent(jobstore.JobWatcher, jobstore.UpdateEvent, jobState)
	})

	// update the job state
	previousState := jobState.State
	jobState.State = request.NewState
	jobState.Version++
	jobState.UpdateTime = b.clock.Now().UTC()

	jobStateData, err := json.Marshal(jobState)
	if err != nil {
		return err
	}

	err = bucket.Put([]byte(request.JobID), jobStateData)
	if err != nil {
		return err
	}

	if request.NewState.IsTerminal() {
		err = b.inProgressIndex.Remove(tx, []byte(request.JobID))
		if err != nil {
			return err
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
	// creating the bucket (jobs_history.JOBID)
	if bkt, err := NewBucketPath(BucketJobsHistory, updateJob.JobID).Get(tx, true); err != nil {
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

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.ExecutionWatcher, jobstore.CreateEvent, execution)
	})

	// Check for the existence of this ID and if it doesn't already exist, then create
	// it
	if bucket, err := NewBucketPath(BucketExecutions).Get(tx, false); err != nil {
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
		newExecution.CreateTime = existingExecution.CreateTime
	}
	if newExecution.UpdateTime.IsZero() {
		newExecution.UpdateTime = b.clock.Now().UTC()
	}
	if newExecution.Version == 0 {
		newExecution.Version = existingExecution.Version + 1
	}

	err = mergo.Merge(&newExecution, existingExecution)
	if err != nil {
		return err
	}

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.ExecutionWatcher, jobstore.UpdateEvent, newExecution)
	})

	data, err := json.Marshal(newExecution)
	if err != nil {
		return err
	}

	bucket, err := NewBucketPath(BucketExecutions).Get(tx, false)
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
	// creating the bucket (executions_history.<jobid>)
	if bkt, err := NewBucketPath(BucketExecutionsHistory, updated.JobID).Get(tx, true); err != nil {
		return err
	} else {
		seq := BucketSequenceString(tx, bkt)
		if err = bkt.Put([]byte(seq), data); err != nil {
			return err
		}
	}

	return nil
}

// CreateEvaluation creates a new evaluation
func (b *BoltJobStore) CreateEvaluation(ctx context.Context, eval models.Evaluation) error {
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.createEvaluation(tx, eval)
	})
}

func (b *BoltJobStore) createEvaluation(tx *bolt.Tx, eval models.Evaluation) error {
	_, err := b.getJob(tx, eval.JobID)
	if err != nil {
		return err
	}

	// If there is no error getting an eval with this ID, then it already exists
	if _, err = b.getEvaluation(tx, eval.ID); err == nil {
		return bacerrors.NewAlreadyExists(eval.ID, "Evaluation")
	}

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.EvaluationWatcher, jobstore.CreateEvent, eval)
	})

	data, err := json.Marshal(eval)
	if err != nil {
		return err
	}

	if bkt, err := NewBucketPath(BucketEvaluations).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put([]byte(eval.ID), data); err != nil {
			return err
		}
	}

	return nil
}

// GetEvaluation retrieves the specified evaluation
func (b *BoltJobStore) GetEvaluation(ctx context.Context, id string) (models.Evaluation, error) {
	var eval models.Evaluation
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		eval, err = b.getEvaluation(tx, id)
		return
	})

	return eval, err
}

func (b *BoltJobStore) getEvaluation(tx *bolt.Tx, id string) (models.Evaluation, error) {
	var eval models.Evaluation

	if bkt, err := NewBucketPath(BucketEvaluations).Get(tx, false); err != nil {
		return eval, err
	} else {
		data := bkt.Get([]byte(id))
		if data == nil {
			return eval, bacerrors.NewEvaluationNotFound(id)
		}

		err = json.Unmarshal(data, &eval)
		if err != nil {
			return eval, err
		}
	}

	return eval, nil
}

// DeleteEvaluation deletes the specified evaluation
func (b *BoltJobStore) DeleteEvaluation(ctx context.Context, id string) error {
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.deleteEvaluation(tx, id)
	})
}

func (b *BoltJobStore) deleteEvaluation(tx *bolt.Tx, id string) error {
	eval, err := b.getEvaluation(tx, id)
	if err != nil {
		return err
	}

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.EvaluationWatcher, jobstore.DeleteEvent, eval)
	})

	if bkt, err := NewBucketPath(BucketEvaluations).Get(tx, false); err != nil {
		return err
	} else {
		err := bkt.Delete([]byte(id))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *BoltJobStore) Close(ctx context.Context) error {
	for _, w := range b.watchers {
		w.Close()
	}

	log.Ctx(ctx).Debug().Msg("closing bolt-backed job store")
	return b.database.Close()
}
