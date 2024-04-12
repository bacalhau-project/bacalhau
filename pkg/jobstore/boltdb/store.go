package boltjobstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/benbjohnson/clock"
	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	bolt "go.etcd.io/bbolt"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	BucketJobs             = "jobs"
	BucketJobExecutions    = "executions"
	BucketJobEvaluations   = "evaluations"
	BucketJobHistory       = "job_history"
	BucketExecutionHistory = "execution_history"

	BucketTagsIndex        = "idx_tags"        // tag -> Job id
	BucketProgressIndex    = "idx_inprogress"  // job-id -> {}
	BucketNamespacesIndex  = "idx_namespaces"  // namespace -> Job id
	BucketExecutionsIndex  = "idx_executions"  // execution-id -> Job id
	BucketEvaluationsIndex = "idx_evaluations" // evaluation-id -> Job id
)

var SpecKey = []byte("spec")

type BoltJobStore struct {
	database    *bolt.DB
	clock       clock.Clock
	marshaller  marshaller.Marshaller
	watchers    []*jobstore.Watcher
	watcherLock sync.Mutex

	inProgressIndex  *Index
	namespacesIndex  *Index
	tagsIndex        *Index
	executionsIndex  *Index
	evaluationsIndex *Index
}

type Option func(store *BoltJobStore)

func WithClock(clock clock.Clock) Option {
	return func(store *BoltJobStore) {
		store.clock = clock
	}
}

// NewBoltJobStore creates a new job store where data is held in buckets,
// and indexed by special [Index] instances, also backed by buckets.
// Data is currently structured as followed
//
// bucket Jobs
//
//	bucket jobID
//		key    spec
//		key state -> state
//		bucket executions -> key executionID -> Execution
//		bucket execution_history -> key  []sequence -> History
//		bucket job_history -> key  []sequence -> History
//		bucket evaluations -> key executionID -> Execution
//
// Indexes are structured as :
//
//	TagsIndex        = tag -> Job id
//	ProgressIndex    = job-id -> {}
//	NamespacesIndex  = namespace -> Job id
//	ExecutionsIndex  = execution-id -> Job id
//	EvaluationsIndex = evaluation-id -> Job id
func NewBoltJobStore(dbPath string, options ...Option) (*BoltJobStore, error) {
	db, err := GetDatabase(dbPath)
	if err != nil {
		return nil, err
	}

	store := &BoltJobStore{
		database:   db,
		clock:      clock.New(),
		marshaller: marshaller.NewJSONMarshaller(),
		watchers:   make([]*jobstore.Watcher, 0), //nolint:gomnd
	}

	for _, opt := range options {
		opt(store)
	}

	// Create the top level buckets ready for use as they
	// will definitely be required
	err = db.Update(func(tx *bolt.Tx) (err error) {
		// Create the top level jobs bucket, and the
		_, err = tx.CreateBucketIfNotExists([]byte(BucketJobs))
		if err != nil {
			return err
		}

		indexBuckets := []string{
			BucketTagsIndex,
			BucketProgressIndex,
			BucketNamespacesIndex,
			BucketExecutionsIndex,
			BucketEvaluationsIndex,
		}
		for _, ib := range indexBuckets {
			_, err = tx.CreateBucketIfNotExists([]byte(ib))
			if err != nil {
				return err
			}
		}

		return nil
	})

	store.inProgressIndex = NewIndex(BucketProgressIndex)
	store.namespacesIndex = NewIndex(BucketNamespacesIndex)
	store.tagsIndex = NewIndex(BucketTagsIndex)
	store.executionsIndex = NewIndex(BucketExecutionsIndex)
	store.evaluationsIndex = NewIndex(BucketEvaluationsIndex)

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
func (b *BoltJobStore) GetJob(ctx context.Context, id string) (models.Job, error) {
	var job models.Job
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		job, err = b.getJob(tx, id)
		return
	})
	return job, err
}

func (b *BoltJobStore) getJob(tx *bolt.Tx, jobID string) (models.Job, error) {
	var job models.Job

	jobID, err := b.reifyJobID(tx, jobID)
	if err != nil {
		return job, err
	}

	data := GetBucketData(tx, NewBucketPath(BucketJobs, jobID), SpecKey)
	if data == nil {
		return job, bacerrors.NewJobNotFound(jobID)
	}

	err = b.marshaller.Unmarshal(data, &job)
	return job, err
}

// reifyJobID ensures the provided job ID is a full-length ID. This is either through
// returning the ID, or resolving the short ID to a single job id.
func (b *BoltJobStore) reifyJobID(tx *bolt.Tx, jobID string) (string, error) {
	if idgen.ShortUUID(jobID) == jobID {
		bktJobs, err := NewBucketPath(BucketJobs).Get(tx, false)
		if err != nil {
			return "", err
		}

		found := make([]string, 0, 1)

		cursor := bktJobs.Cursor()
		prefix := []byte(jobID)
		for k, _ := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = cursor.Next() {
			found = append(found, string(k))
		}

		switch len(found) {
		case 0:
			return "", bacerrors.NewJobNotFound(jobID)
		case 1:
			return found[0], nil
		default:
			return "", bacerrors.NewMultipleJobsFound(jobID, found)
		}
	}

	// Return what we were given
	return jobID, nil
}

func (b *BoltJobStore) getExecution(tx *bolt.Tx, id string) (models.Execution, error) {
	var exec models.Execution

	key, err := b.getExecutionJobID(tx, id)
	if err != nil {
		return exec, err
	}

	if bkt, err := NewBucketPath(BucketJobs, key, BucketJobExecutions).Get(tx, false); err != nil {
		return exec, err
	} else {
		data := bkt.Get([]byte(id))
		if data == nil {
			return exec, jobstore.NewErrExecutionNotFound(id)
		}

		err = b.marshaller.Unmarshal(data, &exec)
		if err != nil {
			return exec, err
		}
	}

	return exec, nil
}

func (b *BoltJobStore) getExecutionJobID(tx *bolt.Tx, id string) (string, error) {
	keys, err := b.executionsIndex.List(tx, []byte(id))
	if err != nil {
		return "", err
	}

	if len(keys) != 1 {
		return "", fmt.Errorf("too many leaf nodes in execution index")
	}

	return string(keys[0]), nil
}

func (b *BoltJobStore) getExecutions(tx *bolt.Tx, options jobstore.GetExecutionsOptions) ([]models.Execution, error) {
	jobID, err := b.reifyJobID(tx, options.JobID)
	if err != nil {
		return nil, err
	}

	// load latest job state if requested
	var job *models.Job
	if options.IncludeJob {
		j, err := b.getJob(tx, options.JobID)
		if err != nil {
			return nil, err
		}
		job = &j
	}

	bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobExecutions).Get(tx, false)
	if err != nil {
		return nil, err
	}

	var execs []models.Execution

	err = bkt.ForEach(func(_ []byte, v []byte) error {
		var es models.Execution
		err = b.marshaller.Unmarshal(v, &es)
		if err != nil {
			return err
		}

		es.Job = job
		execs = append(execs, es)
		return nil
	})

	return execs, err
}

func (b *BoltJobStore) jobExists(tx *bolt.Tx, jobID string) bool {
	_, err := b.getJob(tx, jobID)
	return err == nil
}

// GetJobs returns all Jobs that match the provided query
func (b *BoltJobStore) GetJobs(ctx context.Context, query jobstore.JobQuery) (*jobstore.JobQueryResponse, error) {
	var response *jobstore.JobQueryResponse
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		response, err = b.getJobs(tx, query)
		return
	})
	return response, err
}

func (b *BoltJobStore) getJobs(tx *bolt.Tx, query jobstore.JobQuery) (*jobstore.JobQueryResponse, error) {
	jobSet, err := b.getJobsInitialSet(tx, query)
	if err != nil {
		return nil, err
	}

	jobSet, err = b.getJobsIncludeTags(tx, jobSet, query.IncludeTags)
	if err != nil {
		return nil, err
	}

	jobSet, err = b.getJobsExcludeTags(tx, jobSet, query.ExcludeTags)
	if err != nil {
		return nil, err
	}

	result, err := b.getJobsBuildList(tx, jobSet, query)
	if err != nil {
		return nil, err
	}

	// Sort the jobs according to the query.SortBy and query.SortOrder
	listSorter := func(i, j int) bool {
		switch query.SortBy {
		case "modified_at":
			if query.SortReverse {
				return result[i].ModifyTime > result[j].ModifyTime
			} else {
				return result[i].ModifyTime < result[j].ModifyTime
			}
		default:
			// We apply created_at as a default sort so that we can use it for pagination.
			// Without a known default we won't have a stable sort that makes sense for
			// offsets/limits.
			if query.SortReverse {
				return result[i].CreateTime > result[j].CreateTime
			} else {
				return result[i].CreateTime < result[j].CreateTime
			}
		}
	}
	sort.Slice(result, listSorter)

	// If we have a selector, filter the results to only those that match
	if query.Selector != nil {
		var filtered []models.Job
		for _, job := range result {
			if query.Selector.Matches(labels.Set(job.Labels)) {
				filtered = append(filtered, job)
			}
		}
		result = filtered
	}

	jobs, more := b.getJobsWithinLimit(result, query)

	response := &jobstore.JobQueryResponse{
		Jobs:   jobs,
		Offset: query.Offset,
		Limit:  query.Limit,
	}

	// If we don't have 'limit' jobs, then there definitely aren't any more
	if more {
		response.NextOffset = query.Offset + query.Limit
	}

	return response, nil
}

// getJobsWithinLimit returns the initial set of jobs to be considered for GetJobs response.
// It either returns all jobs, or jobs for a specific client if specified in the query.
func (b *BoltJobStore) getJobsInitialSet(tx *bolt.Tx, query jobstore.JobQuery) (map[string]struct{}, error) {
	jobSet := make(map[string]struct{})

	if query.ReturnAll || query.Namespace == "" {
		bkt, err := NewBucketPath(BucketJobs).Get(tx, false)
		if err != nil {
			return nil, err
		}

		err = bkt.ForEachBucket(func(k []byte) error {
			jobSet[string(k)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		ids, err := b.namespacesIndex.List(tx, []byte(query.Namespace))
		if err != nil {
			return nil, err
		}

		for _, k := range ids {
			jobSet[string(k)] = struct{}{}
		}
	}

	return jobSet, nil
}

// getJobsIncludeTags filters out jobs that don't have ANY of the tags specified in the query.
func (b *BoltJobStore) getJobsIncludeTags(tx *bolt.Tx, jobSet map[string]struct{}, tags []string) (map[string]struct{}, error) {
	if len(tags) == 0 {
		return jobSet, nil
	}
	tagSet := make(map[string]struct{})
	for _, tag := range tags {
		tagLabel := []byte(strings.ToLower(tag))
		ids, err := b.tagsIndex.List(tx, tagLabel)
		if err != nil {
			return nil, err
		}

		for _, k := range ids {
			tagSet[string(k)] = struct{}{}
		}
	}

	// remove jobs that are not in the tag set
	for k := range jobSet {
		if _, ok := tagSet[k]; !ok {
			delete(jobSet, k)
		}
	}

	return jobSet, nil
}

// getJobsExcludeTags filters out jobs that have ANY of the tags specified in the query.
func (b *BoltJobStore) getJobsExcludeTags(tx *bolt.Tx, jobSet map[string]struct{}, tags []string) (map[string]struct{}, error) {
	if len(tags) == 0 {
		return jobSet, nil
	}

	for _, tag := range tags {
		tagLabel := []byte(strings.ToLower(tag))
		ids, err := b.tagsIndex.List(tx, tagLabel)
		if err != nil {
			return nil, err
		}

		for _, k := range ids {
			delete(jobSet, string(k))
		}
	}

	return jobSet, nil
}

func (b *BoltJobStore) getJobsBuildList(tx *bolt.Tx, jobSet map[string]struct{}, query jobstore.JobQuery) ([]models.Job, error) {
	var result []models.Job

	for key := range jobSet {
		var job models.Job

		path := NewBucketPath(BucketJobs, key)
		data := GetBucketData(tx, path, SpecKey)
		err := b.marshaller.Unmarshal(data, &job)
		if err != nil {
			return nil, err
		}
		result = append(result, job)
	}

	listSorter := b.getListSorter(result, query)
	sort.Slice(result, listSorter)

	return result, nil
}

func (b *BoltJobStore) getJobsWithinLimit(jobs []models.Job, query jobstore.JobQuery) ([]models.Job, bool) {
	if query.Offset >= uint32(len(jobs)) {
		return []models.Job{}, false
	}

	jobsFiltered := jobs[query.Offset:]
	if query.Limit == 0 {
		return jobsFiltered, false
	}

	limit := math.Min(uint32(len(jobsFiltered)), query.Limit)
	filteredLength := uint32(len(jobsFiltered))

	jobsFiltered = jobsFiltered[:limit]

	return jobsFiltered, filteredLength > query.Limit
}

func (b *BoltJobStore) getListSorter(jobs []models.Job, query jobstore.JobQuery) func(i, j int) bool {
	return func(i, j int) bool {
		switch query.SortBy {
		case "id":
			if query.SortReverse {
				// what does it mean to sort by ID?
				return jobs[i].ID > jobs[j].ID
			} else {
				return jobs[i].ID < jobs[j].ID
			}
		case "created_at":
			if query.SortReverse {
				return jobs[i].CreateTime > jobs[j].CreateTime
			} else {
				return jobs[i].CreateTime < jobs[j].CreateTime
			}
		default:
			return false
		}
	}
}

// GetExecutions returns the current job state for the provided job id
func (b *BoltJobStore) GetExecutions(ctx context.Context, options jobstore.GetExecutionsOptions) ([]models.Execution, error) {
	var state []models.Execution

	err := b.database.View(func(tx *bolt.Tx) (err error) {
		state, err = b.getExecutions(tx, options)
		return
	})

	return state, err
}

// GetInProgressJobs gets a list of the currently in-progress jobs, if a job type is supplied then
// only jobs of that type will be retrieved
func (b *BoltJobStore) GetInProgressJobs(ctx context.Context, jobType string) ([]models.Job, error) {
	var infos []models.Job
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		infos, err = b.getInProgressJobs(tx, jobType)
		return
	})
	return infos, err
}

func (b *BoltJobStore) getInProgressJobs(tx *bolt.Tx, jobType string) ([]models.Job, error) {
	var infos []models.Job
	var keys [][]byte

	keys, err := b.inProgressIndex.List(tx)
	if err != nil {
		return nil, err
	}

	for _, jobIDKey := range keys {
		k, typ := splitInProgressIndexKey(string(jobIDKey))
		if jobType != "" && jobType != typ {
			// If the user supplied a job type to filter on, and it doesn't match the job type
			// then skip this job
			continue
		}

		job, err := b.getJob(tx, k)
		if err != nil {
			return nil, err
		}
		infos = append(infos, job)
	}

	return infos, nil
}

// splitInProgressIndexKey returns the job type and the job index from
// the in-progress index key. If no delimiter is found, then this index
// was created before this feature was implemented, and we are unable
// to filter on its type so will return "" as the type.
func splitInProgressIndexKey(key string) (string, string) {
	parts := strings.Split(key, ":")
	if len(parts) == 1 {
		return key, ""
	}

	k, typ := parts[1], parts[0]
	return k, typ
}

// createInProgressIndexKey will create a composite key for the in-progress index
func createInProgressIndexKey(job *models.Job) string {
	return fmt.Sprintf("%s:%s", job.Type, job.ID)
}

// GetJobHistory returns the job (and execution) history for the provided options
func (b *BoltJobStore) GetJobHistory(ctx context.Context,
	jobID string,
	options jobstore.JobHistoryFilterOptions) ([]models.JobHistory, error) {
	var history []models.JobHistory
	err := b.database.View(func(tx *bolt.Tx) (err error) {
		history, err = b.getJobHistory(tx, jobID, options)
		return
	})

	return history, err
}

func (b *BoltJobStore) getJobHistory(tx *bolt.Tx, jobID string,
	options jobstore.JobHistoryFilterOptions) ([]models.JobHistory, error) {
	var history []models.JobHistory

	jobID, err := b.reifyJobID(tx, jobID)
	if err != nil {
		return history, err
	}

	if !options.ExcludeJobLevel {
		if bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobHistory).Get(tx, false); err != nil {
			return nil, err
		} else {
			err = bkt.ForEach(func(key []byte, data []byte) error {
				var item models.JobHistory

				err := b.marshaller.Unmarshal(data, &item)
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
		if bkt, err := NewBucketPath(BucketJobs, jobID, BucketExecutionHistory).Get(tx, false); err != nil {
			return nil, err
		} else {
			err = bkt.ForEach(func(key []byte, data []byte) error {
				var item models.JobHistory

				err := b.marshaller.Unmarshal(data, &item)
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

	// Filter out anything before the specified Since time, and anything that doesn't match the
	// specified ExecutionID or NodeID
	history = lo.Filter(history, func(event models.JobHistory, index int) bool {
		if options.ExecutionID != "" && !strings.HasPrefix(event.ExecutionID, options.ExecutionID) {
			return false
		}

		if options.NodeID != "" && !strings.HasPrefix(event.NodeID, options.NodeID) {
			return false
		}

		if event.Time.Unix() < options.Since {
			return false
		}
		return true
	})

	sort.Slice(history, func(i, j int) bool { return history[i].Time.UTC().Before(history[j].Time.UTC()) })

	return history, nil
}

// CreateJob creates a new record of a job in the data store
func (b *BoltJobStore) CreateJob(ctx context.Context, job models.Job, event models.Event) error {
	job.State = models.NewJobState(models.JobStateTypePending)
	job.Revision = 1
	job.CreateTime = b.clock.Now().UTC().UnixNano()
	job.ModifyTime = b.clock.Now().UTC().UnixNano()
	job.Normalize()
	err := job.Validate()
	if err != nil {
		return err
	}
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.createJob(tx, job, event)
	})
}

func (b *BoltJobStore) createJob(tx *bolt.Tx, job models.Job, event models.Event) error {
	if b.jobExists(tx, job.ID) {
		return jobstore.NewErrJobAlreadyExists(job.ID)
	}

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.JobWatcher, jobstore.CreateEvent, job)
	})

	jobIDKey := []byte(job.ID)
	if bkt, err := NewBucketPath(BucketJobs, job.ID).Get(tx, true); err != nil {
		return err
	} else {
		// Create the evaluations and executions buckets and so forth
		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketJobExecutions)); err != nil {
			return err
		}
		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketJobEvaluations)); err != nil {
			return err
		}
		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketJobHistory)); err != nil {
			return err
		}

		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketExecutionHistory)); err != nil {
			return err
		}
	}

	// Write the job to the Job bucket
	jobData, err := b.marshaller.Marshal(job)
	if err != nil {
		return err
	}

	if bkt, err := NewBucketPath(BucketJobs, job.ID).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put(SpecKey, jobData); err != nil {
			return err
		}
	}

	// Create a composite key for the in progress index
	jobkey := createInProgressIndexKey(&job)
	if err = b.inProgressIndex.Add(tx, []byte(jobkey)); err != nil {
		return err
	}

	if err = b.namespacesIndex.Add(tx, jobIDKey, []byte(job.Namespace)); err != nil {
		return err
	}

	// Write sentinels keys for specific tags
	for tag := range job.Labels {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Add(tx, jobIDKey, tagBytes); err != nil {
			return err
		}
	}

	return b.appendJobHistory(tx, job, models.JobStateTypePending, event)
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

	// Delete the Job bucket (and everything within it)
	if bkt, err := NewBucketPath(BucketJobs).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.DeleteBucket([]byte(jobID)); err != nil {
			return err
		}
	}

	// We'll remove the job from the in progress index using just it's ID in case
	// it predates when we switched to composite keys.
	_ = b.inProgressIndex.Remove(tx, []byte(job.ID))

	compositeKey := createInProgressIndexKey(&job)
	if err = b.inProgressIndex.Remove(tx, []byte(compositeKey)); err != nil {
		return err
	}

	if err = b.namespacesIndex.Remove(tx, jobIDKey, []byte(job.Namespace)); err != nil {
		return err
	}

	// Delete sentinels keys for specific tags
	for tag := range job.Labels {
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
	bucket, err := NewBucketPath(BucketJobs, request.JobID).Get(tx, true)
	if err != nil {
		return err
	}

	job, err := b.getJob(tx, request.JobID)
	if err != nil {
		return err
	}

	// check the expected state
	if err := request.Condition.Validate(job); err != nil {
		return err
	}

	if job.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(request.JobID, job.State.StateType, request.NewState)
	}

	// Setup an oncommit handler after the obvious errors/checks
	tx.OnCommit(func() {
		b.triggerEvent(jobstore.JobWatcher, jobstore.UpdateEvent, job)
	})

	// update the job state
	previousState := job.State.StateType
	job.State.StateType = request.NewState
	job.State.Message = request.Event.Message
	job.Revision++
	job.ModifyTime = b.clock.Now().UTC().UnixNano()

	jobStateData, err := b.marshaller.Marshal(job)
	if err != nil {
		return err
	}

	// Re-write the state
	err = bucket.Put(SpecKey, jobStateData)
	if err != nil {
		return err
	}

	if job.IsTerminal() {
		// Remove the job from the in progress index, first checking for legacy items
		// and then removing the composite.  Once we are confident no legacy items
		// are left in the old index we can stick to just the composite
		_ = b.inProgressIndex.Remove(tx, []byte(job.ID))

		composite := createInProgressIndexKey(&job)
		err = b.inProgressIndex.Remove(tx, []byte(composite))
		if err != nil {
			return err
		}
	}

	return b.appendJobHistory(tx, job, previousState, request.Event)
}

func (b *BoltJobStore) appendJobHistory(tx *bolt.Tx, updateJob models.Job, previousState models.JobStateType, event models.Event) error {
	historyEntry := models.JobHistory{
		Type:  models.JobHistoryTypeJobLevel,
		JobID: updateJob.ID,
		JobState: &models.StateChange[models.JobStateType]{
			Previous: previousState,
			New:      updateJob.State.StateType,
		},
		NewRevision: updateJob.Revision,
		Comment:     event.Message,
		Event:       event,
		Time:        time.Unix(0, updateJob.ModifyTime),
	}
	data, err := b.marshaller.Marshal(historyEntry)
	if err != nil {
		return err
	}

	if bkt, err := NewBucketPath(BucketJobs, updateJob.ID, BucketJobHistory).Get(tx, true); err != nil {
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
func (b *BoltJobStore) CreateExecution(ctx context.Context, execution models.Execution, event models.Event) error {
	if execution.CreateTime == 0 {
		execution.CreateTime = b.clock.Now().UTC().UnixNano()
	}
	if execution.ModifyTime == 0 {
		execution.ModifyTime = execution.CreateTime
	}
	if execution.Revision == 0 {
		execution.Revision = 1
	}
	// Ensure the job is not included in the execution when persisting it
	execution.Job = nil
	execution.Normalize()
	err := execution.Validate()
	if err != nil {
		return err
	}
	return b.database.Update(func(tx *bolt.Tx) (err error) {
		return b.createExecution(tx, execution, event)
	})
}

func (b *BoltJobStore) createExecution(tx *bolt.Tx, execution models.Execution, event models.Event) error {
	if !b.jobExists(tx, execution.JobID) {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}

	execID := []byte(execution.ID)
	tx.OnCommit(func() {
		b.triggerEvent(jobstore.ExecutionWatcher, jobstore.CreateEvent, execution)
	})

	// Get the history bucket for this job ID, which involves potentially
	// creating the bucket (jobs/JOBID/job_history)

	// Check for the existence of this ID and if it doesn't already exist, then create
	// it
	if bucket, err := NewBucketPath(BucketJobs, execution.JobID, BucketJobExecutions).Get(tx, true); err != nil {
		return err
	} else {
		_, err = b.getExecution(tx, execution.ID)
		if err == nil {
			return jobstore.NewErrExecutionAlreadyExists(execution.ID)
		}

		if data, err := b.marshaller.Marshal(execution); err != nil {
			return err
		} else {
			err = bucket.Put(execID, data)
			if err != nil {
				return err
			}
		}

		// Add an index for the execution ID to the job id
		if err = b.executionsIndex.Add(tx, []byte(execution.JobID), []byte(execution.ID)); err != nil {
			return err
		}
	}

	return b.appendExecutionHistory(tx, execution, models.ExecutionStateNew, event)
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
	if existingExecution.IsTerminalComputeState() {
		return jobstore.NewErrExecutionAlreadyTerminal(
			request.ExecutionID, existingExecution.ComputeState.StateType, request.NewValues.ComputeState.StateType)
	}

	// populate default values, maintain existing execution createTime
	newExecution := request.NewValues
	newExecution.CreateTime = existingExecution.CreateTime
	if newExecution.ModifyTime == 0 {
		newExecution.ModifyTime = b.clock.Now().UTC().UnixNano()
	}
	if newExecution.Revision == 0 {
		newExecution.Revision = existingExecution.Revision + 1
	}
	newExecution.Normalize()

	err = mergo.Merge(&newExecution, existingExecution)
	if err != nil {
		return err
	}

	tx.OnCommit(func() {
		b.triggerEvent(jobstore.ExecutionWatcher, jobstore.UpdateEvent, newExecution)
	})

	data, err := b.marshaller.Marshal(newExecution)
	if err != nil {
		return err
	}

	bucket, err := NewBucketPath(BucketJobs, newExecution.JobID, BucketJobExecutions).Get(tx, false)
	if err != nil {
		return err
	} else {
		err = bucket.Put([]byte(newExecution.ID), data)
		if err != nil {
			return err
		}
	}

	return b.appendExecutionHistory(tx, newExecution, existingExecution.ComputeState.StateType, request.Event)
}

func (b *BoltJobStore) appendExecutionHistory(tx *bolt.Tx, updated models.Execution,
	previous models.ExecutionStateType, event models.Event) error {
	historyEntry := models.JobHistory{
		Type:        models.JobHistoryTypeExecutionLevel,
		JobID:       updated.JobID,
		NodeID:      updated.NodeID,
		ExecutionID: updated.ID,
		ExecutionState: &models.StateChange[models.ExecutionStateType]{
			Previous: previous,
			New:      updated.ComputeState.StateType,
		},
		NewRevision: updated.Revision,
		Comment:     event.Message,
		Event:       event,
		Time:        time.Unix(0, updated.ModifyTime),
	}

	data, err := b.marshaller.Marshal(historyEntry)
	if err != nil {
		return err
	}

	// Get the history bucket for this job ID, which involves potentially
	// creating the bucket (executions_history.<jobid>)
	if bkt, err := NewBucketPath(BucketJobs, updated.JobID, BucketExecutionHistory).Get(tx, true); err != nil {
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

	data, err := b.marshaller.Marshal(eval)
	if err != nil {
		return err
	}

	if bkt, err := NewBucketPath(BucketJobs, eval.JobID, BucketJobEvaluations).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put([]byte(eval.ID), data); err != nil {
			return err
		}
	}

	// Add an index for the eval pointing to the job id
	err = b.evaluationsIndex.Add(tx, []byte(eval.JobID), []byte(eval.ID))
	if err != nil {
		return err
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

	key, err := b.getEvaluationJobID(tx, id)
	if err != nil {
		return eval, err
	}

	if bkt, err := NewBucketPath(BucketJobs, key, BucketJobEvaluations).Get(tx, false); err != nil {
		return eval, err
	} else {
		data := bkt.Get([]byte(id))
		if data == nil {
			return eval, bacerrors.NewEvaluationNotFound(id)
		}

		err = b.marshaller.Unmarshal(data, &eval)
		if err != nil {
			return eval, err
		}
	}

	return eval, nil
}

func (b *BoltJobStore) getEvaluationJobID(tx *bolt.Tx, id string) (string, error) {
	keys, err := b.evaluationsIndex.List(tx, []byte(id))
	if err != nil {
		return "", err
	}

	if len(keys) != 1 {
		return "", fmt.Errorf("too many leaf nodes in evaluation index")
	}

	return string(keys[0]), nil
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

	jobID, err := b.getEvaluationJobID(tx, id)
	if err != nil {
		return err
	}

	if bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobEvaluations).Get(tx, false); err != nil {
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

// Static check to ensure that BoltJobStore implements jobstore.Store
var _ jobstore.Store = (*BoltJobStore)(nil)
