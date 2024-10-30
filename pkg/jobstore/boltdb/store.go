package boltjobstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/benbjohnson/clock"
	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/analytics"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/boltdblib"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	boltdb_watcher "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	BucketJobs           = "jobs"
	BucketJobExecutions  = "executions"
	BucketJobEvaluations = "evaluations"
	BucketJobHistory     = "history"

	BucketTagsIndex        = "idx_tags"        // tag -> Job id
	BucketProgressIndex    = "idx_inprogress"  // job-id -> {}
	BucketNamespacesIndex  = "idx_namespaces"  // namespace -> Job id
	BucketExecutionsIndex  = "idx_executions"  // execution-id -> Job id
	BucketEvaluationsIndex = "idx_evaluations" // evaluation-id -> Job id

	// Event-related buckets
	eventsBucket      = "v1_events"
	checkpointsBucket = "v1_checkpoints"
)

var SpecKey = []byte("spec")

type BoltJobStore struct {
	database   *bolt.DB
	eventStore *boltdb_watcher.EventStore
	clock      clock.Clock
	marshaller marshaller.Marshaller

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
//		bucket history -> key  []sequence -> History
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
	db, err := boltdblib.Open(dbPath)
	if err != nil {
		return nil, err
	}

	store := &BoltJobStore{
		database:   db,
		clock:      clock.New(),
		marshaller: marshaller.NewJSONMarshaller(),
	}

	for _, opt := range options {
		opt(store)
	}

	// Create the top level buckets ready for use as they
	// will definitely be required
	if err = db.Update(func(tx *bolt.Tx) error {
		// Create the top level jobs bucket
		_, err := tx.CreateBucketIfNotExists([]byte(BucketJobs))
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
			_, err := tx.CreateBucketIfNotExists([]byte(ib))
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	store.inProgressIndex = NewIndex(BucketProgressIndex)
	store.namespacesIndex = NewIndex(BucketNamespacesIndex)
	store.tagsIndex = NewIndex(BucketTagsIndex)
	store.executionsIndex = NewIndex(BucketExecutionsIndex)
	store.evaluationsIndex = NewIndex(BucketEvaluationsIndex)

	eventObjectSerializer := watcher.NewJSONSerializer()
	err = errors.Join(
		eventObjectSerializer.RegisterType(jobstore.EventObjectExecutionUpsert, reflect.TypeOf(jobstore.ExecutionUpsert{})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register event object types: %w", err)
	}

	eventStore, err := boltdb_watcher.NewEventStore(store.database,
		boltdb_watcher.WithEventsBucket(eventsBucket),
		boltdb_watcher.WithCheckpointBucket(checkpointsBucket),
		boltdb_watcher.WithEventSerializer(eventObjectSerializer),
	)
	store.eventStore = eventStore

	return store, err
}

// BeginTx starts a new writable transaction for the store
func (b *BoltJobStore) BeginTx(ctx context.Context) (jobstore.TxContext, error) {
	tx, err := b.database.Begin(true)
	if err != nil {
		return nil, err
	}
	return jobstore.NewTracingContext(boltdblib.NewTxContext(ctx, tx)), nil
}

// GetJob retrieves the Job identified by the id string. If the job isn't found it will
// return an indicating the error.
func (b *BoltJobStore) GetJob(ctx context.Context, id string) (models.Job, error) {
	var job models.Job
	err := boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
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
		return job, jobstore.NewErrJobNotFound(jobID)
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
			return "", NewBoltDBError(err)
		}

		found := make([]string, 0, 1)

		cursor := bktJobs.Cursor()
		prefix := []byte(jobID)
		for k, _ := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = cursor.Next() {
			found = append(found, string(k))
		}

		switch len(found) {
		case 0:
			return "", jobstore.NewErrJobNotFound(jobID)
		case 1:
			return found[0], nil
		default:
			return "", jobstore.NewErrMultipleJobsFound(jobID)
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
		return exec, NewBoltDBError(err)
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
		return "", NewBoltDBError(err)
	}

	if len(keys) != 1 {
		return "", jobstore.NewErrMultipleExecutionsFound(id)
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

	// Sort By Given Order By
	var sortFnc func(a, b models.Execution) int
	switch options.OrderBy {
	// create_time will eventually be deprectated. It is being used for backward compatibility.
	case "create_time", "created_at", "": //nolint: goconst
		sortFnc = func(a, b models.Execution) int { return util.Compare[int64]{}.Cmp(a.CreateTime, b.CreateTime) }
	// modify_time will eventually be deprecated. It is being used for backward compatibility.
	case "modify_time", "modified_at":
		sortFnc = func(a, b models.Execution) int { return util.Compare[int64]{}.Cmp(a.ModifyTime, b.ModifyTime) }
	default:
		return nil, fmt.Errorf("OrderBy %s not supported for getExecutions", options.OrderBy)
	}

	if options.Reverse {
		baseSortFnc := sortFnc
		sortFnc = func(a, b models.Execution) int {
			r := baseSortFnc(a, b)
			if r == -1 {
				return 1
			}
			if r == 1 {
				return -1
			}
			return 0
		}
	}

	bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobExecutions).Get(tx, false)
	if err != nil {
		return nil, NewBoltDBError(err)
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

	// sort executions
	slices.SortFunc(execs, sortFnc)

	// apply limit
	if options.Limit > 0 && len(execs) > options.Limit {
		execs = execs[:options.Limit]
	}

	return execs, err
}

func (b *BoltJobStore) jobExists(tx *bolt.Tx, jobID string) bool {
	_, err := b.getJob(tx, jobID)
	return err == nil
}

// GetJobs returns all Jobs that match the provided query
func (b *BoltJobStore) GetJobs(ctx context.Context, query jobstore.JobQuery) (*jobstore.JobQueryResponse, error) {
	var response *jobstore.JobQueryResponse
	err := boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
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
	var sortFunc func(a, b models.Job) int
	switch query.SortBy {
	case "created_at", "":
		sortFunc = func(a, b models.Job) int { return util.Compare[int64]{}.Cmp(a.CreateTime, b.CreateTime) }
	case "modified_at":
		sortFunc = func(a, b models.Job) int { return util.Compare[int64]{}.Cmp(a.ModifyTime, b.ModifyTime) }
	default:
		return nil, fmt.Errorf("OrderBy %s not supported for listJobs", query.SortBy)
	}
	if query.SortReverse {
		baseSortFnc := sortFunc
		sortFunc = func(a, b models.Job) int {
			r := baseSortFnc(a, b)
			if r == -1 {
				return 1
			}
			if r == 1 {
				return -1
			}
			return 0
		}
	}

	slices.SortFunc(result, sortFunc)

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

	if more {
		response.NextOffset = query.Offset + uint64(query.Limit)
	}

	return response, nil
}

// getJobsInitialSet returns the initial set of jobs to be considered for GetJobs response.
// It either returns all jobs, or jobs for a specific client if specified in the query.
func (b *BoltJobStore) getJobsInitialSet(tx *bolt.Tx, query jobstore.JobQuery) (map[string]struct{}, error) {
	jobSet := make(map[string]struct{})

	if query.ReturnAll || query.Namespace == "" {
		bkt, err := NewBucketPath(BucketJobs).Get(tx, false)
		if err != nil {
			return nil, NewBoltDBError(err)
		}

		err = bkt.ForEachBucket(func(k []byte) error {
			jobSet[string(k)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, NewBoltDBError(err)
		}
	} else {
		ids, err := b.namespacesIndex.List(tx, []byte(query.Namespace))
		if err != nil {
			return nil, NewBoltDBError(err)
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
			return nil, NewBoltDBError(err)
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
			return nil, NewBoltDBError(err)
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

// TODO: Ignoring linting for now. Fixing this is a task by itself
//
//nolint:gosec // G115: slice length is always non-negative in Go
func (b *BoltJobStore) getJobsWithinLimit(jobs []models.Job, query jobstore.JobQuery) ([]models.Job, bool) {
	if query.Offset >= uint64(len(jobs)) {
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

	err := boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		state, err = b.getExecutions(tx, options)
		return
	})

	return state, err
}

// GetInProgressJobs gets a list of the currently in-progress jobs, if a job type is supplied then
// only jobs of that type will be retrieved
func (b *BoltJobStore) GetInProgressJobs(ctx context.Context, jobType string) ([]models.Job, error) {
	var infos []models.Job
	err := boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
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
		return nil, NewBoltDBError(err)
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

// GetJobHistory retrieves the paginated job history for a given job ID based on the specified query.
//
// This method performs a read transaction on the Bolt DB and fetches the job history
// for the specified jobID. It supports pagination by processing an offset and limit
// defined either in the query or via a `NextToken`. Pagination tokens help in fetching
// the next set of results if the query returns a partial result due to the limit.
//
// Pagination Behavior:
//   - The `NextToken` in the query allows the caller to continue fetching subsequent pages.
//   - If the result set reaches the limit specified in the query, a new `NextToken` is generated.
//   - If no records are found in the current query, but the job or execution is not in a terminal state,
//     the same `NextToken` will be returned to indicate that more history might still be available in the future.
//   - Pagination only stops when there are no more records to fetch *and* the job/execution is in a terminal state.
func (b *BoltJobStore) GetJobHistory(ctx context.Context,
	jobID string,
	query jobstore.JobHistoryQuery,
) (*jobstore.JobHistoryQueryResponse, error) {
	var response *jobstore.JobHistoryQueryResponse
	err := boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		response, err = b.getJobHistory(tx, jobID, query)
		return
	})
	return response, err
}

func (b *BoltJobStore) getJobHistory(tx *bolt.Tx, jobID string, query jobstore.JobHistoryQuery) (*jobstore.JobHistoryQueryResponse, error) {
	jobID, err := b.reifyJobID(tx, jobID)
	if err != nil {
		return nil, err
	}

	offset, limit, err := b.parseHistoryPaginationParams(query)
	if err != nil {
		return nil, err
	}

	bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobHistory).Get(tx, false)
	if err != nil {
		// If the bucket doesn't exist, then we return an empty response to maintain compatibility
		// with < v1.5.0 versions as the history bucket name was renamed in v1.5.0 without migration
		// as migration is not worth the complexity
		if errors.Is(err, bolt.ErrBucketNotFound) {
			return &jobstore.JobHistoryQueryResponse{}, nil
		}
		return nil, NewBoltDBError(err)
	}

	var history []models.JobHistory
	var lastSeq uint64

	cursor := bkt.Cursor()
	for k, v := cursor.Seek(uint64ToBytes(offset)); k != nil; k, v = cursor.Next() {
		var item models.JobHistory
		if err := b.marshaller.Unmarshal(v, &item); err != nil {
			return nil, err
		}

		if b.filterHistoryItem(item, query) {
			history = append(history, item)
			lastSeq = bytesToUint64(k)
		}

		//nolint:gosec // G115: history within reasonable bounds
		if uint32(len(history)) == limit {
			break
		}
	}

	response := &jobstore.JobHistoryQueryResponse{
		JobHistory: history,
	}

	// Determine if we should continue pagination
	shouldContinue, err := b.shouldContinueHistoryPagination(tx, jobID, cursor, query)
	if err != nil {
		return nil, err
	}

	if shouldContinue {
		newOffset := lastSeq + 1
		if len(history) == 0 {
			// If we didn't find any items, then we need to continue from the last offset
			newOffset = offset
		}
		response.NextToken = models.NewPagingToken(&models.PagingTokenParams{
			Offset: newOffset,
			Limit:  query.Limit,
		}).String()
	}

	return response, nil
}

func (b *BoltJobStore) parseHistoryPaginationParams(query jobstore.JobHistoryQuery) (uint64, uint32, error) {
	const defaultTokenLimit = 100
	offset := uint64(0)
	limit := uint32(defaultTokenLimit)

	if query.NextToken != "" {
		token, err := models.NewPagingTokenFromString(query.NextToken)
		if err != nil {
			return 0, 0, jobstore.NewBadRequestError(fmt.Sprintf("invalid next token: %s", err))
		}
		offset = token.Offset
		if token.Limit != 0 {
			limit = token.Limit
		}
	}

	if query.Limit != 0 {
		limit = query.Limit
	}

	return offset, limit, nil
}

// filterHistoryItem filter out anything before the specified Since time,
// and anything that doesn't match the specified ExecutionID
func (b *BoltJobStore) filterHistoryItem(item models.JobHistory, query jobstore.JobHistoryQuery) bool {
	if query.ExecutionID != "" && !strings.HasPrefix(item.ExecutionID, query.ExecutionID) {
		return false
	}
	if item.Time.Unix() < query.Since {
		return false
	}
	if query.ExcludeJobLevel && item.Type == models.JobHistoryTypeJobLevel {
		return false
	}
	if query.ExcludeExecutionLevel && item.Type == models.JobHistoryTypeExecutionLevel {
		return false
	}
	return true
}

func (b *BoltJobStore) shouldContinueHistoryPagination(
	tx *bolt.Tx,
	jobID string,
	cursor *bolt.Cursor,
	query jobstore.JobHistoryQuery,
) (bool, error) {
	// If there are more items in the bucket, then we should continue
	if k, _ := cursor.Next(); k != nil {
		return true, nil
	}

	// Otherwise, we need to check if the job or execution are in a terminal state
	// For execution level events, stop if the execution in terminal state
	if query.ExecutionID != "" {
		execution, err := b.getExecution(tx, query.ExecutionID)
		if err != nil {
			return false, err
		}
		return !execution.IsTerminalState(), nil
	}

	// If querying all executions or job level events, stop if the job is in terminal state
	job, err := b.getJob(tx, jobID)
	if err != nil {
		return false, err
	}
	return !job.IsTerminal(), nil
}

// CreateJob creates a new record of a job in the data store
func (b *BoltJobStore) CreateJob(ctx context.Context, job models.Job) error {
	job.State = models.NewJobState(models.JobStateTypePending)
	job.Revision = 1
	job.CreateTime = b.clock.Now().UTC().UnixNano()
	job.ModifyTime = b.clock.Now().UTC().UnixNano()
	job.Normalize()
	err := job.Validate()
	if err != nil {
		return jobstore.NewJobStoreError(err.Error())
	}
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.createJob(tx, job)
	})
}

func (b *BoltJobStore) createJob(tx *bolt.Tx, job models.Job) error {
	if b.jobExists(tx, job.ID) {
		return jobstore.NewErrJobAlreadyExists(job.ID)
	}

	jobIDKey := []byte(job.ID)
	if bkt, err := NewBucketPath(BucketJobs, job.ID).Get(tx, true); err != nil {
		return NewBoltDBError(err)
	} else {
		// Create the evaluations and executions buckets and so forth
		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketJobExecutions)); err != nil {
			return err
		}
		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketJobEvaluations)); err != nil {
			return NewBoltDBError(err)
		}
		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketJobHistory)); err != nil {
			return NewBoltDBError(err)
		}
	}

	// Write the job to the Job bucket
	jobData, err := b.marshaller.Marshal(job)
	if err != nil {
		return err
	}

	if bkt, err := NewBucketPath(BucketJobs, job.ID).Get(tx, false); err != nil {
		return NewBoltDBError(err)
	} else {
		if err = bkt.Put(SpecKey, jobData); err != nil {
			return err
		}
	}

	// Create a composite key for the in progress index
	jobkey := createInProgressIndexKey(&job)
	if err = b.inProgressIndex.Add(tx, []byte(jobkey)); err != nil {
		return NewBoltDBError(err)
	}

	if err = b.namespacesIndex.Add(tx, jobIDKey, []byte(job.Namespace)); err != nil {
		return NewBoltDBError(err)
	}

	// Write sentinels keys for specific tags
	for tag := range job.Labels {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Add(tx, jobIDKey, tagBytes); err != nil {
			return err
		}
	}

	return nil
}

// DeleteJob removes the specified job from the system entirely
func (b *BoltJobStore) DeleteJob(ctx context.Context, jobID string) error {
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.deleteJob(tx, jobID)
	})
}

func (b *BoltJobStore) deleteJob(tx *bolt.Tx, jobID string) error {
	jobIDKey := []byte(jobID)

	job, err := b.getJob(tx, jobID)
	if err != nil {
		if bacerrors.IsError(err) {
			return err
		}
		return NewBoltDBError(err)
	}

	// Delete the Job bucket (and everything within it)
	if bkt, err := NewBucketPath(BucketJobs).Get(tx, false); err != nil {
		return NewBoltDBError(err)
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
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
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

	// update the job state
	job.State.StateType = request.NewState
	job.State.Message = request.Message
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
		tx.OnCommit(func() {
			// TODO to include execution telemetry
			analytics.EmitEvent(context.TODO(), analytics.NewJobTerminalEvent(job))
		})
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

	return nil
}

// AddJobHistory appends a new history entry to the job history
func (b *BoltJobStore) AddJobHistory(ctx context.Context, jobID string, events ...models.Event) error {
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		for _, event := range events {
			if err = b.addJobHistory(tx, jobID, event); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BoltJobStore) addJobHistory(tx *bolt.Tx, jobID string, event models.Event) error {
	return b.addHistory(tx, jobID, models.JobHistory{
		Type:  models.JobHistoryTypeJobLevel,
		JobID: jobID,
		Event: event,
		Time:  b.clock.Now().UTC(),
	})
}

func (b *BoltJobStore) addExecutionHistory(tx *bolt.Tx, jobID, executionID string, event models.Event) error {
	return b.addHistory(tx, jobID, models.JobHistory{
		Type:        models.JobHistoryTypeExecutionLevel,
		JobID:       jobID,
		ExecutionID: executionID,
		Event:       event,
		Time:        b.clock.Now().UTC(),
	})
}

func (b *BoltJobStore) addHistory(tx *bolt.Tx, jobID string, historyEntry models.JobHistory) error {
	bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobHistory).Get(tx, false)
	if err != nil {
		return err
	}

	seq, err := bkt.NextSequence()
	if err != nil {
		return err
	}

	historyEntry.SeqNum = seq
	data, err := b.marshaller.Marshal(historyEntry)
	if err != nil {
		return err
	}

	return bkt.Put(uint64ToBytes(seq), data)
}

// CreateExecution creates a record of a new execution
func (b *BoltJobStore) CreateExecution(ctx context.Context, execution models.Execution) error {
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
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.createExecution(tx, execution)
	})
}

func (b *BoltJobStore) createExecution(tx *bolt.Tx, execution models.Execution) error {
	if !b.jobExists(tx, execution.JobID) {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}

	execID := []byte(execution.ID)

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

		if err = b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
			Operation:  watcher.OperationCreate,
			ObjectType: jobstore.EventObjectExecutionUpsert,
			Object:     jobstore.ExecutionUpsert{Current: &execution},
		}); err != nil {
			return err
		}
	}

	tx.OnCommit(func() {
		analytics.EmitEvent(context.TODO(), analytics.NewCreatedExecutionEvent(execution))
	})
	return nil
}

// UpdateExecution updates the state of a single execution by loading from storage,
// updating and then writing back in a single transaction
func (b *BoltJobStore) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) error {
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
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

	if err = b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
		Operation:  watcher.OperationUpdate,
		ObjectType: jobstore.EventObjectExecutionUpsert,
		Object: jobstore.ExecutionUpsert{
			Current: &newExecution, Previous: &existingExecution,
		},
	}); err != nil {
		return err
	}

	tx.OnCommit(func() {
		if newExecution.IsTerminalState() {
			analytics.EmitEvent(context.TODO(), analytics.NewTerminalExecutionEvent(newExecution))
		}
		if newExecution.IsDiscarded() {
			analytics.EmitEvent(context.TODO(), analytics.NewComputeMessageExecutionEvent(newExecution))
		}
	})

	return nil
}

// AddExecutionHistory appends a new history entry to the execution history
func (b *BoltJobStore) AddExecutionHistory(ctx context.Context, jobID, executionID string, events ...models.Event) error {
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		for _, event := range events {
			if err = b.addExecutionHistory(tx, jobID, executionID, event); err != nil {
				return err
			}
		}
		return nil
	})
}

// CreateEvaluation creates a new evaluation
func (b *BoltJobStore) CreateEvaluation(ctx context.Context, eval models.Evaluation) error {
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
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
		return jobstore.NewErrEvaluationAlreadyExists(eval.ID)
	}

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

	return b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: jobstore.EventObjectEvaluation,
		Object:     &eval,
	})
}

// GetEvaluation retrieves the specified evaluation
func (b *BoltJobStore) GetEvaluation(ctx context.Context, id string) (models.Evaluation, error) {
	var eval models.Evaluation
	err := boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
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
			return eval, jobstore.NewErrEvaluationNotFound(id)
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
		return "", jobstore.NewErrMultipleEvaluationsFound(id)
	}

	return string(keys[0]), nil
}

// DeleteEvaluation deletes the specified evaluation
func (b *BoltJobStore) DeleteEvaluation(ctx context.Context, id string) error {
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.deleteEvaluation(tx, id)
	})
}

func (b *BoltJobStore) deleteEvaluation(tx *bolt.Tx, id string) error {
	eval, err := b.getEvaluation(tx, id)
	if err != nil {
		return err
	}

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

	return b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
		Operation:  watcher.OperationDelete,
		ObjectType: jobstore.EventObjectEvaluation,
		Object:     &eval,
	})
}

// GetEventStore returns the event store
func (b *BoltJobStore) GetEventStore() watcher.EventStore {
	return b.eventStore
}

func (b *BoltJobStore) Close(ctx context.Context) error {
	log.Ctx(ctx).Debug().Msg("closing bolt-backed job store")
	var mErr error
	mErr = errors.Join(mErr, b.eventStore.Close(ctx))
	mErr = errors.Join(mErr, b.database.Close())
	return mErr
}

// Static check to ensure that BoltJobStore implements jobstore.Store
var _ jobstore.Store = (*BoltJobStore)(nil)
