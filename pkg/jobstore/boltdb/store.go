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
	"github.com/samber/lo"
	bolt "go.etcd.io/bbolt"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
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
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	BucketJobs           = "jobs"
	BucketJobExecutions  = "executions"
	BucketJobEvaluations = "evaluations"
	BucketJobHistory     = "history"
	BucketJobVersions    = "versions" // bucket for job versions

	BucketTagsIndex                 = "idx_tags"                  // tag -> Job id
	BucketProgressIndex             = "idx_inprogress"            // job-id -> {}
	BucketNamespacesIndex           = "idx_namespaces"            // namespace -> Job id
	BucketExecutionsIndex           = "idx_executions"            // execution-id -> Job id
	BucketEvaluationsIndex          = "idx_evaluations"           // evaluation-id -> Job id
	BucketJobsNamesIndex            = "idx_job_names"             // job-name -> Job id
	BucketInProgressExecutionsIndex = "idx_inprogress_executions" // executionID:jobID -> {}
	BucketExecutionsByNodeIndex     = "idx_executions_by_node"    // node-id -> executionID:jobID

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

	inProgressIndex           *Index
	namespacesIndex           *Index
	tagsIndex                 *Index
	executionsIndex           *Index
	evaluationsIndex          *Index
	namesIndex                *Index
	inProgressExecutionsIndex *Index
	executionsByNodeIndex     *Index
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
			BucketJobsNamesIndex,
			BucketInProgressExecutionsIndex,
			BucketExecutionsByNodeIndex,
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
	store.namesIndex = NewIndex(BucketJobsNamesIndex)
	store.inProgressExecutionsIndex = NewIndex(BucketInProgressExecutionsIndex)
	store.executionsByNodeIndex = NewIndex(BucketExecutionsByNodeIndex)

	eventObjectSerializer := watcher.NewJSONSerializer()
	err = errors.Join(
		eventObjectSerializer.RegisterType(jobstore.EventObjectExecutionUpsert, reflect.TypeOf(models.ExecutionUpsert{})),
		eventObjectSerializer.RegisterType(jobstore.EventObjectEvaluation, reflect.TypeOf(models.Evaluation{})),
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

// metricRecorder returns a new metric recorder with the given attributes
func (b *BoltJobStore) metricRecorder(
	ctx context.Context, bucket, operation string, attrs ...attribute.KeyValue) *telemetry.MetricRecorder {
	recorder := telemetry.NewMetricRecorder(
		append(attrs,
			semconv.DBSystemKey.String("boltdb"),
			semconv.DBNamespaceKey.String("jobstore"),
			semconv.DBOperationName(operation),
			semconv.DBCollectionName(bucket),
		)...,
	)
	recorder.Count(ctx, jobstore.OperationCount)
	return recorder
}

// BeginTx starts a new writable transaction for the store
func (b *BoltJobStore) BeginTx(ctx context.Context) (jobstore.TxContext, error) {
	tx, err := b.database.Begin(true)
	if err != nil {
		return nil, err
	}
	return boltdblib.NewTracingContext(boltdblib.NewTxContext(ctx, tx)), nil
}

// GetJob retrieves the Job identified by the id string. If the job isn't found it will
// return an indicating the error.
func (b *BoltJobStore) GetJob(ctx context.Context, id string) (job models.Job, err error) {
	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationGet)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		job, err = b.getJob(ctx, tx, recorder, id)
		return
	})
	return job, err
}

func (b *BoltJobStore) getJob(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string) (models.Job, error) {
	var job models.Job

	jobID, err := b.reifyJobID(ctx, tx, recorder, jobID)
	if err != nil {
		return job, err
	}

	data := GetBucketData(tx, NewBucketPath(BucketJobs, jobID), SpecKey)
	if data == nil {
		return job, jobstore.NewErrJobNotFound(jobID)
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)

	err = b.marshaller.Unmarshal(data, &job)
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartUnmarshal)
	recorder.CountN(ctx, jobstore.DataRead, int64(len(data)))
	recorder.Count(ctx, jobstore.RowsRead)
	return job, err
}

// reifyJobID ensures the provided job ID is a full-length ID. This is either through
// returning the ID, or resolving the short ID to a single job id.
func (b *BoltJobStore) reifyJobID(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string) (string, error) {
	if idgen.ShortUUID(jobID) == jobID {
		defer recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartReifyID)

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

// getExecution retrieves the Execution identified by the execution ID only, which requires
// an index call to find the job ID first.
func (b *BoltJobStore) getExecution(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, execID string) (models.Execution, error) {
	var exec models.Execution

	// Get the job ID from the index
	jobID, err := b.getExecutionJobID(tx, execID)
	if err != nil {
		return exec, err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexRead)

	// Now get the execution with the job ID and execution ID
	return b.getExecutionWithJobID(ctx, tx, recorder, jobID, execID)
}

func (b *BoltJobStore) getExecutionWithJobID(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID, execID string) (models.Execution, error) {
	var exec models.Execution

	if bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobExecutions).Get(tx, false); err != nil {
		return exec, NewBoltDBError(err)
	} else {
		data := bkt.Get([]byte(execID))
		if data == nil {
			return exec, jobstore.NewErrExecutionNotFound(execID)
		}
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)
		recorder.CountN(ctx, jobstore.DataRead, int64(len(data)))
		recorder.Count(ctx, jobstore.RowsRead)

		err = b.marshaller.Unmarshal(data, &exec)
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartUnmarshal)
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

func (b *BoltJobStore) getExecutions(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	options jobstore.GetExecutionsOptions) ([]models.Execution, error) {
	// Get execution IDs based on query type
	executions, err := b.getExecutionsForQuery(ctx, tx, recorder, options)
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return []models.Execution{}, nil
	}

	// Prepare for execution filtering and loading
	// cache for loaded jobs and versions to be included in executions if options.IncludeJob is true
	loadedJobs := make(map[string]models.Job)
	// cache for latest version of each job to filter executions if not all or specific version  requested
	latestVersions := make(map[string]uint64)
	// set of node IDs for faster lookups
	nodeSet := lo.SliceToMap(options.NodeIDs, func(nodeID string) (string, struct{}) {
		return nodeID, struct{}{}
	})

	// helper function to generate job version keys
	jobVersionKey := func(jobID string, version uint64) string {
		return fmt.Sprintf("%s::%d", jobID, version)
	}

	// Convert execution IDs to executions with filtering
	var execs []models.Execution
	for _, execution := range executions {
		var latestJobVersion uint64
		if options.JobVersion == 0 && !options.AllJobVersions {
			if v, ok := latestVersions[execution.JobID]; ok {
				latestJobVersion = v
			} else {
				latestJobVersion, err = b.getLatestJobVersion(ctx, tx, recorder, execution.JobID)
				if err != nil {
					return []models.Execution{}, err
				}
				latestVersions[execution.JobID] = latestJobVersion
			}
		}

		// Apply all filters
		if b.filterJobExecutionItem(execution, options, nodeSet, latestJobVersion) {
			// Add job if requested
			if options.IncludeJob {
				loadedJob, ok := loadedJobs[jobVersionKey(execution.JobID, execution.JobVersion)]
				if !ok {
					// Load job if we don't have it cached
					loadedJob, err = b.getJobVersion(ctx, tx, recorder, execution.JobID, execution.JobVersion)
					if err != nil {
						return nil, err
					}
					loadedJobs[jobVersionKey(execution.JobID, execution.JobVersion)] = loadedJob
				}
				execution.Job = &loadedJob
			}
			execs = append(execs, execution)
		}
	}

	// Sort executions
	b.sortExecutions(ctx, recorder, options, execs)

	// Apply limit
	if options.Limit > 0 && len(execs) > options.Limit {
		execs = execs[:options.Limit]
	}

	return execs, nil
}

// getExecutionsForQuery gets execution IDs based on the query parameters
func (b *BoltJobStore) getExecutionsForQuery(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	options jobstore.GetExecutionsOptions) ([]models.Execution, error) {
	// If JobID is specified, get executions for that job
	if options.JobID != "" {
		return b.getExecutionsForJob(ctx, tx, recorder, options.JobID)
	}

	// Otherwise use indexes
	executionPairs, err := b.getExecutionIDsFromIndexes(ctx, tx, recorder, options)
	if err != nil {
		return nil, err
	}
	execs := make([]models.Execution, 0, len(executionPairs))
	for _, execPair := range executionPairs {
		exec, err := b.getExecutionWithJobID(ctx, tx, recorder, execPair.JobID, execPair.ExecutionID)
		if err != nil {
			if errors.Is(err, jobstore.NewErrExecutionNotFound(execPair.ExecutionID)) {
				log.Warn().Str("execution_id", execPair.ExecutionID).Msg("Execution in index but not found in store")
				continue // Skip missing executions
			}
			return nil, err
		}
		execs = append(execs, exec)
	}

	return execs, nil
}

// getExecutionsForJob gets all executions for a specific job.
func (b *BoltJobStore) getExecutionsForJob(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string) ([]models.Execution, error) {
	jobID, err := b.reifyJobID(ctx, tx, recorder, jobID)
	if err != nil {
		return nil, err
	}

	bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobExecutions).Get(tx, false)
	if err != nil {
		return nil, NewBoltDBError(err)
	}

	var executions []models.Execution
	err = bkt.ForEach(func(k []byte, data []byte) error {
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)
		recorder.CountN(ctx, jobstore.DataRead, int64(len(data)))
		recorder.Count(ctx, jobstore.RowsRead)

		var exec models.Execution
		err = b.marshaller.Unmarshal(data, &exec)
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartUnmarshal)
		if err != nil {
			return err
		}
		executions = append(executions, exec)
		return nil
	})

	return executions, err
}

// getExecutionIDsFromIndexes retrieves execution ID and job ID pairs that match ALL specified filter options (AND relationship)
func (b *BoltJobStore) getExecutionIDsFromIndexes(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	options jobstore.GetExecutionsOptions) ([]ExecutionJobPair, error) {
	var executionIDSets []map[string]struct{}

	// Get execution IDs from node index if NodeIDs specified
	if len(options.NodeIDs) > 0 {
		nodeExecutionIDs := make(map[string]struct{})
		for _, nodeID := range options.NodeIDs {
			compositeKeys, err := b.executionsByNodeIndex.List(tx, []byte(nodeID))
			if err != nil {
				return nil, NewBoltDBError(err)
			}
			for _, compositeKey := range compositeKeys {
				nodeExecutionIDs[string(compositeKey)] = struct{}{}
			}
		}
		executionIDSets = append(executionIDSets, nodeExecutionIDs)
		recorder.Latency(ctx, jobstore.OperationPartDuration, "index_read_by_node")
	}

	// Get execution IDs from in-progress index if InProgressOnly specified
	if options.InProgressOnly {
		inProgressExecutionIDs := make(map[string]struct{})
		compositeKeys, err := b.inProgressExecutionsIndex.List(tx)
		if err != nil {
			return nil, NewBoltDBError(err)
		}
		for _, compositeKey := range compositeKeys {
			inProgressExecutionIDs[string(compositeKey)] = struct{}{}
		}
		executionIDSets = append(executionIDSets, inProgressExecutionIDs)
		recorder.Latency(ctx, jobstore.OperationPartDuration, "index_read_in_progress")
	}

	// If no indexes were used, return empty
	if len(executionIDSets) == 0 {
		return []ExecutionJobPair{}, nil
	}

	// Find intersection of all sets
	result := executionIDSets[0]
	for i := 1; i < len(executionIDSets); i++ {
		intersection := make(map[string]struct{})
		for compositeKey := range result {
			if _, exists := executionIDSets[i][compositeKey]; exists {
				intersection[compositeKey] = struct{}{}
			}
		}
		result = intersection
	}

	// Convert map to slice and decode composite keys
	var executionPairs []ExecutionJobPair
	for compositeKey := range result {
		executionID, jobID, err := decodeExecutionJobKey(compositeKey)
		if err != nil {
			log.Warn().Str("composite_key", compositeKey).Err(err).Msg("Failed to decode composite key")
			continue // Skip invalid keys
		}
		executionPairs = append(executionPairs, ExecutionJobPair{
			ExecutionID: executionID,
			JobID:       jobID,
		})
	}

	return executionPairs, nil
}

func (b *BoltJobStore) sortExecutions(ctx context.Context, recorder *telemetry.MetricRecorder,
	options jobstore.GetExecutionsOptions,
	execs []models.Execution,
) {
	var sortFnc func(a, b models.Execution) int
	switch options.OrderBy {
	// create_time will eventually be deprecated. It is being used for backward compatibility.
	case "create_time", "created_at", "": //nolint: goconst
		sortFnc = func(a, b models.Execution) int { return util.Compare[int64]{}.Cmp(a.CreateTime, b.CreateTime) }
	// modify_time will eventually be deprecated. It is being used for backward compatibility.
	case "modify_time", "modified_at":
		sortFnc = func(a, b models.Execution) int { return util.Compare[int64]{}.Cmp(a.ModifyTime, b.ModifyTime) }
	default:
		// This should be handled by the caller, but for safety:
		return
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

	slices.SortFunc(execs, sortFnc)
	recorder.Latency(ctx, jobstore.OperationPartDuration, "sort")
}

func (b *BoltJobStore) jobExists(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string) bool {
	_, err := b.getJob(ctx, tx, recorder, jobID)
	return err == nil
}

// jobExistsByName checks if a job with the specified name exists in the given namespace
func (b *BoltJobStore) jobExistsByName(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, name, namespace string) bool {
	_, err := b.getJobByName(ctx, tx, recorder, name, namespace)
	return err == nil
}

// GetJobs returns all Jobs that match the provided query
func (b *BoltJobStore) GetJobs(
	ctx context.Context, query jobstore.JobQuery) (response *jobstore.JobQueryResponse, err error) {
	scope := jobstore.AttrScopeAll
	if query.Namespace != "" && !query.ReturnAll {
		scope = jobstore.AttrScopeNamespace
	}
	attrs := []attribute.KeyValue{
		jobstore.AttrScopeKey.String(scope),
		jobstore.AttrNamespaceKey.String(query.Namespace),
	}
	if len(query.IncludeTags) > 0 {
		attrs = append(attrs, attribute.Bool("query.include_tags", true))
	}
	if len(query.ExcludeTags) > 0 {
		attrs = append(attrs, attribute.Bool("query.exclude_tags", true))
	}
	if query.Selector != nil {
		attrs = append(attrs, attribute.Bool("query.selector", true))
	}
	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationList, attrs...)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		response, err = b.getJobs(ctx, tx, recorder, query)
		return
	})
	return response, err
}

func (b *BoltJobStore) getJobs(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	query jobstore.JobQuery) (*jobstore.JobQueryResponse, error) {
	jobSet, err := b.getJobsInitialSet(ctx, tx, recorder, query)
	if err != nil {
		return nil, err
	}

	jobSet, err = b.getJobsIncludeTags(ctx, tx, recorder, jobSet, query.IncludeTags)
	if err != nil {
		return nil, err
	}

	jobSet, err = b.getJobsExcludeTags(ctx, tx, recorder, jobSet, query.ExcludeTags)
	if err != nil {
		return nil, err
	}

	result, err := b.getJobsBuildList(ctx, tx, recorder, jobSet, query)
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
	recorder.Latency(ctx, jobstore.OperationPartDuration, "sort")

	// If we have a selector, filter the results to only those that match
	if query.Selector != nil {
		var filtered []models.Job
		for _, job := range result {
			if query.Selector.Matches(labels.Set(job.Labels)) {
				filtered = append(filtered, job)
			}
		}
		result = filtered
		recorder.Latency(ctx, jobstore.OperationPartDuration, "filter_selector")
	}

	jobs, more := b.getJobsWithinLimit(result, query)
	recorder.Latency(ctx, jobstore.OperationPartDuration, "filter_limit")

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

func (b *BoltJobStore) getJobIDByJobName(
	ctx context.Context,
	tx *bolt.Tx,
	recorder *telemetry.MetricRecorder,
	jobNameKey string,
) (string, error) {
	jobIDs, err := b.namesIndex.List(tx, []byte(jobNameKey))
	if err != nil {
		return "", err
	}

	if len(jobIDs) == 0 {
		return "", jobstore.NewErrJobNameIndexNotFound(jobNameKey)
	}

	if len(jobIDs) != 1 {
		return "", jobstore.NewErrMultipleJobIDsForSameJobNameFound(jobNameKey)
	}

	return string(jobIDs[0]), nil
}

// getJobsInitialSet returns the initial set of jobs to be considered for GetJobs response.
// It either returns all jobs, or jobs for a specific client if specified in the query.
func (b *BoltJobStore) getJobsInitialSet(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	query jobstore.JobQuery) (map[string]struct{}, error) {
	jobSet := make(map[string]struct{})
	defer recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexRead)

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
func (b *BoltJobStore) getJobsIncludeTags(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	jobSet map[string]struct{}, tags []string) (map[string]struct{}, error) {
	if len(tags) == 0 {
		return jobSet, nil
	}
	defer recorder.Latency(ctx, jobstore.OperationPartDuration, "filter_include_tags")
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
func (b *BoltJobStore) getJobsExcludeTags(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	jobSet map[string]struct{}, tags []string) (map[string]struct{}, error) {
	if len(tags) == 0 {
		return jobSet, nil
	}
	defer recorder.Latency(ctx, jobstore.OperationPartDuration, "filter_exclude_tags")
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

func (b *BoltJobStore) getJobsBuildList(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	jobSet map[string]struct{}, query jobstore.JobQuery) ([]models.Job, error) {
	var result []models.Job

	defer recorder.Latency(ctx, jobstore.OperationPartDuration, "build_list")

	for key := range jobSet {
		var job models.Job

		path := NewBucketPath(BucketJobs, key)
		data := GetBucketData(tx, path, SpecKey)

		recorder.CountN(ctx, jobstore.DataRead, int64(len(data)))
		recorder.Count(ctx, jobstore.RowsRead)

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
func (b *BoltJobStore) GetExecutions(
	ctx context.Context, options jobstore.GetExecutionsOptions) (state []models.Execution, err error) {
	var attrs []attribute.KeyValue
	if options.IncludeJob {
		attrs = append(attrs, attribute.Bool("query.include_job", true))
	}
	recorder := b.metricRecorder(ctx, BucketJobExecutions, jobstore.AttrOperationList, attrs...)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	if err = options.Validate(); err != nil {
		recorder.Error(err)
		return nil, err
	}

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		state, err = b.getExecutions(ctx, tx, recorder, options)
		return
	})

	return state, err
}

// GetInProgressJobs gets a list of the currently in-progress jobs, if a job type is supplied then
// only jobs of that type will be retrieved
func (b *BoltJobStore) GetInProgressJobs(ctx context.Context, jobType string) (jobs []models.Job, err error) {
	attrs := []attribute.KeyValue{
		jobstore.AttrScopeKey.String(jobstore.AttrScopeInProgress),
	}
	if jobType != "" {
		attrs = append(attrs, attribute.String("query.job_type", jobType))
	}

	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationList, attrs...)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		jobs, err = b.getInProgressJobs(ctx, tx, recorder, jobType)
		return
	})
	return jobs, err
}

func (b *BoltJobStore) getInProgressJobs(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobType string) ([]models.Job, error) {
	var infos []models.Job
	var keys [][]byte

	keys, err := b.inProgressIndex.List(tx)
	if err != nil {
		return nil, NewBoltDBError(err)
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexRead)

	for _, jobIDKey := range keys {
		k, typ := splitInProgressIndexKey(string(jobIDKey))
		if jobType != "" && jobType != typ {
			// If the user supplied a job type to filter on, and it doesn't match the job type
			// then skip this job
			continue
		}

		job, err := b.getJob(ctx, tx, recorder, k)
		if err != nil {
			return nil, err
		}
		infos = append(infos, job)
	}
	return infos, nil
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
) (response *jobstore.JobHistoryQueryResponse, err error) {
	recorder := b.metricRecorder(ctx, BucketJobHistory, jobstore.AttrOperationList,
		jobstore.AttrScopeKey.String(jobstore.AttrScopeJob))
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		response, err = b.getJobHistory(ctx, tx, recorder, jobID, query)
		return
	})
	return response, err
}

func (b *BoltJobStore) getJobHistory(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	jobID string, query jobstore.JobHistoryQuery) (*jobstore.JobHistoryQueryResponse, error) {
	jobID, err := b.reifyJobID(ctx, tx, recorder, jobID)
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
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)

		var item models.JobHistory
		if err := b.marshaller.Unmarshal(v, &item); err != nil {
			return nil, err
		}
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartUnmarshal)
		recorder.CountN(ctx, jobstore.DataRead, int64(len(v)))
		recorder.Count(ctx, jobstore.RowsRead)

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
	shouldContinue, err := b.shouldContinueHistoryPagination(ctx, tx, recorder, jobID, cursor, query)
	if err != nil {
		return nil, err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, "determine_pagination")

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
	// If the requested job version is zero, return only the latest version items
	if query.JobVersion == 0 {
		// We explicitly are including all job versions
		if query.AllJobVersions {
			return true
		}

		// Only include latest job version
		return item.JobVersion == query.LatestJobVersion
	}

	if item.JobVersion != query.JobVersion {
		return false
	}

	return true
}

// filterJobExecutionItem applies filtering logic to an execution based on the provided query options.
// It returns true if the execution should be included in the result set, or false if it should be filtered out.
// The filters applied are, in order:
//   - If JobVersion is specified, only executions with that version are included.
//   - If AllJobVersions is not set, only executions with the latest job version are included.
//   - If Namespace is specified, only executions in that namespace are included.
//   - If InProgressOnly is set, only executions not in a terminal state are included.
//   - If NodeIDs are specified, only executions on those nodes are included.
func (b *BoltJobStore) filterJobExecutionItem(
	item models.Execution,
	query jobstore.GetExecutionsOptions,
	nodeSet map[string]struct{},
	latestJobVersion uint64,
) bool {
	// filter by job version if specified
	if query.JobVersion > 0 && item.JobVersion != query.JobVersion {
		return false
	}

	// filter by latest job version if not all versions are requested
	if query.JobVersion == 0 && !query.AllJobVersions && item.JobVersion != latestJobVersion {
		return false
	}

	// filter by namespace if specified
	if query.Namespace != "" && item.Namespace != query.Namespace {
		return false
	}

	// filter by execution state if specified
	if query.InProgressOnly && item.IsTerminalState() {
		return false
	}

	// filter by node ID if specified
	if len(nodeSet) > 0 && !lo.HasKey(nodeSet, item.NodeID) {
		return false
	}

	return true
}

func (b *BoltJobStore) shouldContinueHistoryPagination(
	ctx context.Context,
	tx *bolt.Tx,
	recorder *telemetry.MetricRecorder,
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
		execution, err := b.getExecutionWithJobID(ctx, tx, recorder, jobID, query.ExecutionID)
		if err != nil {
			return false, err
		}
		return !execution.IsTerminalState(), nil
	}

	// If querying all executions or job level events, stop if the job is in terminal state
	job, err := b.getJob(ctx, tx, recorder, jobID)
	if err != nil {
		return false, err
	}
	return !job.IsTerminal(), nil
}

// CreateJob creates a new record of a job in the data store
func (b *BoltJobStore) CreateJob(ctx context.Context, job models.Job) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationCreate)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	job.State = models.NewJobState(models.JobStateTypePending)
	job.Revision = 1
	job.Version = 1
	job.CreateTime = b.clock.Now().UTC().UnixNano()
	job.ModifyTime = b.clock.Now().UTC().UnixNano()
	job.Normalize()
	err = job.Validate()
	if err != nil {
		return jobstore.NewJobStoreError(err.Error())
	}
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.createJob(ctx, tx, recorder, job)
	})
}

func (b *BoltJobStore) createJob(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, job models.Job) error {
	if b.jobExists(ctx, tx, recorder, job.ID) {
		return jobstore.NewErrJobAlreadyExists(job.ID)
	}

	// Check if a job with this name already exists in the namespace
	if b.jobExistsByName(ctx, tx, recorder, job.Name, job.Namespace) {
		return jobstore.NewErrJobNameAlreadyExists(job.Name, job.Namespace)
	}

	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartValidate)

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
		// Create the versions bucket for storing job versions
		if _, err := bkt.CreateBucketIfNotExists([]byte(BucketJobVersions)); err != nil {
			return NewBoltDBError(err)
		}
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartBucketWrite)

	// Write the job to the Job bucket
	jobData, err := b.marshaller.Marshal(job)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartMarshal)
	recorder.CountN(ctx, jobstore.DataWritten, int64(len(jobData)))

	if bkt, err := NewBucketPath(BucketJobs, job.ID).Get(tx, false); err != nil {
		return NewBoltDBError(err)
	} else {
		// Write the current job spec
		if err = bkt.Put(SpecKey, jobData); err != nil {
			return err
		}

		// Store the initial version in the version bucket
		versionBkt, err := NewBucketPath(BucketJobs, job.ID, BucketJobVersions).Get(tx, false)
		if err != nil {
			return NewBoltDBError(err)
		}

		// Use the job's Version as the key for the version, and update current version
		if err = versionBkt.Put(uint64ToBytes(job.Version), jobData); err != nil {
			return err
		}
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)

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

	// Add job name to the job names index bucket
	jobNameKey := createJobNameIndexKey(job.Name, job.Namespace)
	if err = b.namesIndex.Add(tx, jobIDKey, []byte(jobNameKey)); err != nil {
		return NewBoltDBError(err)
	}

	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexWrite)

	return nil
}

// DeleteJob removes the specified job from the system entirely
func (b *BoltJobStore) DeleteJob(ctx context.Context, jobID string) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationDelete)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.deleteJob(ctx, tx, jobID, recorder)
	})
}

func (b *BoltJobStore) deleteJob(
	ctx context.Context, tx *bolt.Tx, jobID string, recorder *telemetry.MetricRecorder) error {
	jobIDKey := []byte(jobID)

	job, err := b.getJob(ctx, tx, recorder, jobID)
	if err != nil {
		if bacerrors.IsError(err) {
			return err
		}
		return NewBoltDBError(err)
	}

	// Get all executions for this job before deleting the job
	executions, err := b.getExecutions(ctx, tx, recorder, jobstore.GetExecutionsOptions{JobID: jobID})
	if err != nil {
		return err
	}

	// Clean up execution indexes
	for _, execution := range executions {
		compositeKey := encodeExecutionJobKey(execution.ID, execution.JobID)

		// Remove from node index
		if err = b.executionsByNodeIndex.Remove(tx, []byte(compositeKey), []byte(execution.NodeID)); err != nil {
			return err
		}

		// Remove from in-progress index if it was in progress
		if !execution.IsTerminalState() {
			if err = b.inProgressExecutionsIndex.Remove(tx, []byte(compositeKey)); err != nil {
				return err
			}
		}
	}

	// Delete the Job bucket (and everything within it)
	if bkt, err := NewBucketPath(BucketJobs).Get(tx, false); err != nil {
		return NewBoltDBError(err)
	} else {
		if err = bkt.DeleteBucket([]byte(jobID)); err != nil {
			return err
		}
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartBucketDelete)

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

	// Remove from job names bucket
	jobNameKey := createJobNameIndexKey(job.Name, job.Namespace)
	if err = b.namesIndex.Remove(tx, jobIDKey, []byte(jobNameKey)); err != nil {
		return NewBoltDBError(err)
	}

	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexDelete)

	return nil
}

// UpdateJob updates an existing job in the data store
// Only specific fields are updated, and the current job is saved as a new version,
// and the job state is updated to pending.
func (b *BoltJobStore) UpdateJob(ctx context.Context, job models.Job) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationUpdate)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	// Ensure the job has a valid ID
	if job.ID == "" {
		return jobstore.NewJobStoreError("cannot update job without an ID")
	}

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.updateJob(ctx, tx, recorder, job)
	})
}

//nolint:funlen
func (b *BoltJobStore) updateJob(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, updatedJob models.Job) error {
	// Get the existing job
	existingJob, err := b.getJob(ctx, tx, recorder, updatedJob.ID)
	if err != nil {
		return err
	}

	// If name has changed, ensure the new name doesn't already exist
	if updatedJob.Name != existingJob.Name || updatedJob.Namespace != existingJob.Namespace {
		return jobstore.NewJobStoreError("cannot change job name or namespace during update")
	}

	// Verify that this job exists in the names bucket
	jobNameKey := createJobNameIndexKey(existingJob.Name, existingJob.Namespace)
	indexedJobID, err := b.getJobIDByJobName(ctx, tx, recorder, jobNameKey)

	if err != nil {
		if !bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError) {
			return NewBoltDBError(err)
		}

		// If the job isn't in the names index bucket, add it now since it could an old Job
		log.Ctx(ctx).Warn().
			Str("job_id", existingJob.ID).
			Str("job_name", existingJob.Name).
			Str("namespace", existingJob.Namespace).
			Msg("Job exists but not found in names index bucket - adding it now")

		if err = b.namesIndex.Add(tx, []byte(existingJob.ID), []byte(jobNameKey)); err != nil {
			return NewBoltDBError(err)
		}
	} else {
		if indexedJobID != existingJob.ID {
			return jobstore.
				NewJobStoreError(
					fmt.Sprintf(
						"inconsistency between the Job name and its ID. Job name %s with ID %s does not match stored job ID %s",
						existingJob.Name,
						existingJob.ID,
						indexedJobID,
					)).
				WithHint("This usually happens if you try to rerun a job, using its ID, that was created before " +
					"version 1.8")
		}
	}

	// Store the existing job in the versions bucket before updating it
	versionBkt, err := NewBucketPath(BucketJobs, existingJob.ID, BucketJobVersions).Get(tx, false)
	if err != nil {
		return NewBoltDBError(err)
	}

	// Save the current version
	existingJobData, err := b.marshaller.Marshal(existingJob)
	if err != nil {
		return err
	}

	if err = versionBkt.Put(uint64ToBytes(existingJob.Version), existingJobData); err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)

	// Update only the specified fields
	existingJob.Priority = updatedJob.Priority
	existingJob.Count = updatedJob.Count
	existingJob.State = models.NewJobState(models.JobStateTypePending)
	existingJob.Constraints = updatedJob.Constraints
	existingJob.Meta = updatedJob.Meta
	existingJob.Labels = updatedJob.Labels
	existingJob.Tasks = updatedJob.Tasks

	// Increment version and update modification time
	existingJob.Version++
	existingJob.ModifyTime = b.clock.Now().UTC().UnixNano()

	// Normalize and validate the updated job
	existingJob.Normalize()
	if err = existingJob.Validate(); err != nil {
		return jobstore.NewJobStoreError(err.Error())
	}

	// Marshal and write the updated job
	updatedJobData, err := b.marshaller.Marshal(existingJob)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartMarshal)
	recorder.CountN(ctx, jobstore.DataWritten, int64(len(updatedJobData)))

	// Get the job bucket
	bucket, err := NewBucketPath(BucketJobs, existingJob.ID).Get(tx, false)
	if err != nil {
		return NewBoltDBError(err)
	}

	// Update the job in the store
	if err = bucket.Put(SpecKey, updatedJobData); err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)

	// Store the new version in the versions bucket
	if err = versionBkt.Put(uint64ToBytes(existingJob.Version), updatedJobData); err != nil {
		return err
	}

	// Create a composite key for the in progress index
	inProgressIndexKey := createInProgressIndexKey(&existingJob)
	if err = b.inProgressIndex.Add(tx, []byte(inProgressIndexKey)); err != nil {
		return NewBoltDBError(err)
	}

	// Update tags index - first remove all existing tags
	jobIDKey := []byte(existingJob.ID)
	for tag := range existingJob.Labels {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Remove(tx, jobIDKey, tagBytes); err != nil {
			return err
		}
	}

	// Then add all new tags
	for tag := range existingJob.Labels {
		tagBytes := []byte(strings.ToLower(tag))
		if err = b.tagsIndex.Add(tx, jobIDKey, tagBytes); err != nil {
			return err
		}
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexWrite)

	return nil
}

// UpdateJobState updates the current state for a single Job, appending an entry to
// the history at the same time
func (b *BoltJobStore) UpdateJobState(ctx context.Context, request jobstore.UpdateJobStateRequest) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationUpdate,
		jobstore.AttrToStateKey.String(request.NewState.String()))
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.updateJobState(ctx, tx, recorder, request)
	})
}

func (b *BoltJobStore) updateJobState(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	request jobstore.UpdateJobStateRequest) error {
	// Add current state to metrics
	job, err := b.getJob(ctx, tx, recorder, request.JobID)
	if err != nil {
		return err
	}

	// Add current state to metrics
	recorder.AddAttributes(jobstore.AttrFromStateKey.String(job.State.StateType.String()))

	// check the expected state
	if err = request.Condition.Validate(job); err != nil {
		return err
	}

	if job.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(request.JobID, job.State.StateType, request.NewState)
	}

	// update the job state
	// For state changes, we don't increment Version
	job.State.StateType = request.NewState
	job.State.Message = request.Message
	job.Revision++
	job.ModifyTime = b.clock.Now().UTC().UnixNano()

	jobStateData, err := b.marshaller.Marshal(job)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartMarshal)
	recorder.CountN(ctx, jobstore.DataWritten, int64(len(jobStateData)))

	bucket, err := NewBucketPath(BucketJobs, request.JobID).Get(tx, true)
	if err != nil {
		return err
	}

	// Update current job spec
	err = bucket.Put(SpecKey, jobStateData)
	if err != nil {
		return err
	}

	// Store the new version in the versions bucket
	versionBkt, err := NewBucketPath(BucketJobs, request.JobID, BucketJobVersions).Get(tx, false)
	if err != nil {
		return NewBoltDBError(err)
	}

	if err = versionBkt.Put(uint64ToBytes(job.Version), jobStateData); err != nil {
		return err
	}

	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)

	if job.IsTerminal() {
		tx.OnCommit(func() {
			// TODO to include execution telemetry
			analytics.Emit(analytics.NewJobTerminalEvent(job))
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
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexWrite)
	}

	return nil
}

// AddJobHistory appends a new history entry to the job history
func (b *BoltJobStore) AddJobHistory(ctx context.Context, jobID string, jobVersion uint64, events ...models.Event) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobHistory, jobstore.AttrOperationCreate,
		jobstore.AttrScopeKey.String(jobstore.AttrScopeJob))
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		for _, event := range events {
			if err = b.addJobHistory(ctx, tx, recorder, jobID, jobVersion, event); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BoltJobStore) addJobHistory(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string, jobVersion uint64, event models.Event) error {
	return b.addHistory(ctx, tx, recorder, jobID, models.JobHistory{
		Type:       models.JobHistoryTypeJobLevel,
		JobID:      jobID,
		JobVersion: jobVersion,
		Event:      event,
		Time:       b.clock.Now().UTC(),
	})
}

func (b *BoltJobStore) addExecutionHistory(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder,
	jobID string, jobVersion uint64, executionID string, events ...*models.Event) error {
	now := b.clock.Now().UTC()
	for _, event := range events {
		if err := b.addHistory(ctx, tx, recorder, jobID, models.JobHistory{
			Type:        models.JobHistoryTypeExecutionLevel,
			JobID:       jobID,
			JobVersion:  jobVersion,
			ExecutionID: executionID,
			Event:       *event,
			Time:        now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *BoltJobStore) addHistory(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string, historyEntry models.JobHistory) error {
	bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobHistory).Get(tx, false)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartBucketRead)

	seq, err := bkt.NextSequence()
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartSequence)

	historyEntry.SeqNum = seq
	data, err := b.marshaller.Marshal(historyEntry)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartMarshal)
	recorder.CountN(ctx, jobstore.DataWritten, int64(len(data)))

	err = bkt.Put(uint64ToBytes(seq), data)
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)
	return err
}

// CreateExecution creates a record of a new execution
func (b *BoltJobStore) CreateExecution(ctx context.Context, execution models.Execution) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobExecutions, jobstore.AttrOperationCreate)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	if execution.CreateTime == 0 {
		execution.CreateTime = b.clock.Now().UTC().UnixNano()
	}
	if execution.ModifyTime == 0 {
		execution.ModifyTime = execution.CreateTime
	}
	if execution.Revision == 0 {
		execution.Revision = 1
	}
	execution.Normalize()
	err = execution.Validate()
	if err != nil {
		return err
	}
	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.createExecution(ctx, tx, recorder, execution)
	})
}

func (b *BoltJobStore) createExecution(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, execution models.Execution) error {
	if !b.jobExists(ctx, tx, recorder, execution.JobID) {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartValidate)

	execID := []byte(execution.ID)

	// Check for existing execution and create bucket if needed
	bucket, err := NewBucketPath(BucketJobs, execution.JobID, BucketJobExecutions).Get(tx, true)
	if err != nil {
		return err
	}

	// Verify no duplicate execution
	_, err = b.getExecutionWithJobID(ctx, tx, recorder, execution.JobID, execution.ID)
	if err == nil {
		return jobstore.NewErrExecutionAlreadyExists(execution.ID)
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)

	// Marshal and write execution
	data, err := b.marshaller.Marshal(execution)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartMarshal)
	recorder.CountN(ctx, jobstore.DataWritten, int64(len(data)))

	if err = bucket.Put(execID, data); err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)

	// Update index
	if err = b.executionsIndex.Add(tx, []byte(execution.JobID), []byte(execution.ID)); err != nil {
		return err
	}

	// Update new indexes - use composite keys
	compositeKey := encodeExecutionJobKey(execution.ID, execution.JobID)
	if err = b.executionsByNodeIndex.Add(tx, []byte(compositeKey), []byte(execution.NodeID)); err != nil {
		return err
	}

	// Add to in-progress index if execution is not terminal - use composite key
	if !execution.IsTerminalState() {
		if err = b.inProgressExecutionsIndex.Add(tx, []byte(compositeKey)); err != nil {
			return err
		}
	}

	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexWrite)

	// Record event
	if err = b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: jobstore.EventObjectExecutionUpsert,
		Object:     models.ExecutionUpsert{Current: &execution},
	}); err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartEventWrite)

	tx.OnCommit(func() {
		analytics.Emit(analytics.NewCreatedExecutionEvent(execution))
	})
	return nil
}

// UpdateExecution updates the state of a single execution by loading from storage,
// updating and then writing back in a single transaction
func (b *BoltJobStore) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobExecutions, jobstore.AttrOperationUpdate)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.updateExecution(ctx, tx, recorder, request)
	})
}

func (b *BoltJobStore) updateExecution(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, request jobstore.UpdateExecutionRequest) error {
	// Get current execution
	existingExecution, err := b.getExecution(ctx, tx, recorder, request.ExecutionID)
	if err != nil {
		return jobstore.NewErrExecutionNotFound(request.ExecutionID)
	}

	// Record state transitions in metrics
	recorder.AddAttributes(
		jobstore.FromDesiredStateKey.String(existingExecution.DesiredState.StateType.String()),
		jobstore.ToDesiredStateKey.String(request.NewValues.DesiredState.StateType.String()),
		jobstore.AttrFromStateKey.String(existingExecution.ComputeState.StateType.String()),
		jobstore.AttrToStateKey.String(request.NewValues.ComputeState.StateType.String()),
	)

	// Validate state transition
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

	if err = mergo.Merge(&newExecution, existingExecution); err != nil {
		return err
	}

	// Marshal and write updated execution
	data, err := b.marshaller.Marshal(newExecution)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartMarshal)
	recorder.CountN(ctx, jobstore.DataWritten, int64(len(data)))

	bucket, err := NewBucketPath(BucketJobs, newExecution.JobID, BucketJobExecutions).Get(tx, false)
	if err != nil {
		return err
	}

	if err = bucket.Put([]byte(newExecution.ID), data); err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)

	if err = b.updateInProgressExecutionsIndex(ctx, tx, existingExecution, newExecution); err != nil {
		return err
	}

	// Add execution history
	if err = b.addExecutionHistory(
		ctx,
		tx,
		recorder,
		newExecution.JobID,
		newExecution.JobVersion,
		newExecution.ID,
		request.Events...,
	); err != nil {
		return err
	}

	// Store event
	if err = b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
		Operation:  watcher.OperationUpdate,
		ObjectType: jobstore.EventObjectExecutionUpsert,
		Object: models.ExecutionUpsert{
			Current: &newExecution, Previous: &existingExecution, Events: request.Events,
		},
	}); err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartEventWrite)

	tx.OnCommit(func() {
		if newExecution.IsTerminalState() {
			analytics.Emit(analytics.NewTerminalExecutionEvent(newExecution))
		}
		if newExecution.IsDiscarded() {
			analytics.Emit(analytics.NewComputeMessageExecutionEvent(newExecution))
		}
	})

	return nil
}

func (b *BoltJobStore) updateInProgressExecutionsIndex(ctx context.Context, tx *bolt.Tx,
	existingExecution, newExecution models.Execution,
) error {
	wasInProgress := !existingExecution.IsTerminalState()
	isInProgress := !newExecution.IsTerminalState()

	compositeKey := encodeExecutionJobKey(newExecution.ID, newExecution.JobID)

	if wasInProgress && !isInProgress {
		// Execution became terminal, remove from in-progress index
		if err := b.inProgressExecutionsIndex.Remove(tx, []byte(compositeKey)); err != nil {
			return err
		}
	} else if !wasInProgress && isInProgress {
		// Execution became non-terminal (unlikely but possible), add to in-progress index
		if err := b.inProgressExecutionsIndex.Add(tx, []byte(compositeKey)); err != nil {
			return err
		}
	}
	return nil
}

// AddExecutionHistory appends a new history entry to the execution history
func (b *BoltJobStore) AddExecutionHistory(
	ctx context.Context,
	jobID string,
	jobVersion uint64,
	executionID string,
	events ...models.Event,
) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobHistory, jobstore.AttrOperationCreate,
		jobstore.AttrScopeKey.String(jobstore.AttrScopeExecution))
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		eventsValues := make([]*models.Event, len(events))
		for i := range events {
			eventsValues[i] = &events[i]
		}
		return b.addExecutionHistory(ctx, tx, recorder, jobID, jobVersion, executionID, eventsValues...)
	})
}

// CreateEvaluation creates a new evaluation
func (b *BoltJobStore) CreateEvaluation(ctx context.Context, eval models.Evaluation) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobEvaluations, jobstore.AttrOperationCreate)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.createEvaluation(ctx, tx, recorder, eval)
	})
}

func (b *BoltJobStore) createEvaluation(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, eval models.Evaluation) error {
	_, err := b.getJob(ctx, tx, recorder, eval.JobID)
	if err != nil {
		return err
	}

	// If there is no error getting an eval with this ID, then it already exists
	if _, err = b.getEvaluation(ctx, tx, recorder, eval.ID); err == nil {
		return jobstore.NewErrEvaluationAlreadyExists(eval.ID)
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartValidate)

	data, err := b.marshaller.Marshal(eval)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartMarshal)
	recorder.CountN(ctx, jobstore.DataWritten, int64(len(data)))

	if bkt, err := NewBucketPath(BucketJobs, eval.JobID, BucketJobEvaluations).Get(tx, false); err != nil {
		return err
	} else {
		if err = bkt.Put([]byte(eval.ID), data); err != nil {
			return err
		}
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartWrite)

	// Add an index for the eval pointing to the job id
	err = b.evaluationsIndex.Add(tx, []byte(eval.JobID), []byte(eval.ID))
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexWrite)

	err = b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: jobstore.EventObjectEvaluation,
		Object:     eval,
	})
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartEventWrite)
	return err
}

// GetEvaluation retrieves the specified evaluation
func (b *BoltJobStore) GetEvaluation(ctx context.Context, id string) (eval models.Evaluation, err error) {
	recorder := b.metricRecorder(ctx, BucketJobEvaluations, jobstore.AttrOperationGet)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		eval, err = b.getEvaluation(ctx, tx, recorder, id)
		return
	})

	return eval, err
}

func (b *BoltJobStore) getEvaluation(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, id string) (models.Evaluation, error) {
	var eval models.Evaluation

	key, err := b.getEvaluationJobID(ctx, tx, recorder, id)
	if err != nil {
		return eval, err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexRead)

	if bkt, err := NewBucketPath(BucketJobs, key, BucketJobEvaluations).Get(tx, false); err != nil {
		return eval, err
	} else {
		data := bkt.Get([]byte(id))
		if data == nil {
			return eval, jobstore.NewErrEvaluationNotFound(id)
		}
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)

		err = b.marshaller.Unmarshal(data, &eval)
		if err != nil {
			return eval, err
		}
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartUnmarshal)
		recorder.CountN(ctx, jobstore.DataRead, int64(len(data)))
		recorder.Count(ctx, jobstore.RowsRead)
	}

	return eval, nil
}

func (b *BoltJobStore) getEvaluationJobID(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, id string) (string, error) {
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
func (b *BoltJobStore) DeleteEvaluation(ctx context.Context, id string) (err error) {
	recorder := b.metricRecorder(ctx, BucketJobEvaluations, jobstore.AttrOperationDelete)
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	return boltdblib.Update(ctx, b.database, func(tx *bolt.Tx) (err error) {
		return b.deleteEvaluation(ctx, tx, recorder, id)
	})
}

func (b *BoltJobStore) deleteEvaluation(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, id string) error {
	eval, err := b.getEvaluation(ctx, tx, recorder, id)
	if err != nil {
		return err
	}

	jobID, err := b.getEvaluationJobID(ctx, tx, recorder, id)
	if err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexRead)

	if bkt, err := NewBucketPath(BucketJobs, jobID, BucketJobEvaluations).Get(tx, false); err != nil {
		return err
	} else {
		err := bkt.Delete([]byte(id))
		if err != nil {
			return err
		}
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartDelete)
	}

	// Remove from evaluation index
	if err = b.evaluationsIndex.Remove(tx, []byte(jobID), []byte(id)); err != nil {
		return err
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartIndexDelete)

	err = b.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
		Operation:  watcher.OperationDelete,
		ObjectType: jobstore.EventObjectEvaluation,
		Object:     eval,
	})
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartEventWrite)
	return err
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

// GetJobByName retrieves a Job identified by its name and namespace. If the job isn't found
// it will return an error indicating that it was not found.
func (b *BoltJobStore) GetJobByName(ctx context.Context, name, namespace string) (job models.Job, err error) {
	recorder := b.metricRecorder(ctx, BucketJobs, jobstore.AttrOperationGet,
		jobstore.AttrScopeKey.String(jobstore.AttrScopeJob),
		attribute.String("name", name),
		attribute.String("namespace", namespace))
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		job, err = b.getJobByName(ctx, tx, recorder, name, namespace)
		return
	})
	return job, err
}

func (b *BoltJobStore) getJobByName(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, name, namespace string) (models.Job, error) {
	var job models.Job

	if namespace == "" {
		namespace = models.DefaultNamespace
	}

	// Create the job name key in the same format as in createJob
	jobNameKey := createJobNameIndexKey(name, namespace)

	// Look up the job ID from the job names index bucket
	indexedJobID, err := b.getJobIDByJobName(ctx, tx, recorder, jobNameKey)
	if err != nil {
		return job, err
	}

	return b.getJob(ctx, tx, recorder, indexedJobID)
}

// GetJobVersion retrieves a specific version of a job by its ID and version number
func (b *BoltJobStore) GetJobVersion(ctx context.Context, jobID string, version uint64) (job models.Job, err error) {
	recorder := b.metricRecorder(ctx, BucketJobVersions, jobstore.AttrOperationGet,
		jobstore.AttrScopeKey.String(jobstore.AttrScopeJob),
		attribute.Int64("version", int64(version))) //nolint:gosec // G115: version within reasonable bounds
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		job, err = b.getJobVersion(ctx, tx, recorder, jobID, version)
		return
	})
	return job, err
}

func (b *BoltJobStore) getJobVersion(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string, version uint64) (models.Job, error) {
	var job models.Job

	jobID, err := b.reifyJobID(ctx, tx, recorder, jobID)
	if err != nil {
		return job, err
	}

	// Get the version from the versions bucket
	versionBkt, err := NewBucketPath(BucketJobs, jobID, BucketJobVersions).Get(tx, false)
	if err != nil {
		return job, NewBoltDBError(err)
	}

	data := versionBkt.Get(uint64ToBytes(version))
	if data == nil {
		return job, jobstore.NewErrJobVersionNotFound(jobID, version)
	}
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)
	recorder.CountN(ctx, jobstore.DataRead, int64(len(data)))
	recorder.Count(ctx, jobstore.RowsRead)

	err = b.marshaller.Unmarshal(data, &job)
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartUnmarshal)

	return job, err
}

// GetJobVersions returns all available versions of a job
func (b *BoltJobStore) GetJobVersions(ctx context.Context, jobID string) (versions []models.Job, err error) {
	recorder := b.metricRecorder(ctx, BucketJobVersions, jobstore.AttrOperationList,
		jobstore.AttrScopeKey.String(jobstore.AttrScopeJob))
	defer recorder.Done(ctx, jobstore.OperationDuration)
	defer recorder.Error(err)

	err = boltdblib.View(ctx, b.database, func(tx *bolt.Tx) (err error) {
		versions, err = b.getJobVersions(ctx, tx, recorder, jobID)
		return
	})
	return versions, err
}

func (b *BoltJobStore) getJobVersions(
	ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string) ([]models.Job, error) {
	var versions []models.Job

	jobID, err := b.reifyJobID(ctx, tx, recorder, jobID)
	if err != nil {
		return nil, err
	}

	// Get the versions bucket
	versionBkt, err := NewBucketPath(BucketJobs, jobID, BucketJobVersions).Get(tx, false)
	if err != nil {
		return nil, NewBoltDBError(err)
	}

	// Iterate through all versions
	err = versionBkt.ForEach(func(k, v []byte) error {
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)
		recorder.CountN(ctx, jobstore.DataRead, int64(len(v)))
		recorder.Count(ctx, jobstore.RowsRead)

		var job models.Job
		err := b.marshaller.Unmarshal(v, &job)
		if err != nil {
			return err
		}
		recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartUnmarshal)

		versions = append(versions, job)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort versions by Version in ascending order
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version < versions[j].Version
	})
	recorder.Latency(ctx, jobstore.OperationPartDuration, "sort")

	return versions, nil
}

// getLatestJobVersion retrieves the latest version of a job by its ID without loading the full job object.
func (b *BoltJobStore) getLatestJobVersion(ctx context.Context, tx *bolt.Tx, recorder *telemetry.MetricRecorder, jobID string) (
	version uint64, err error) {
	jobID, err = b.reifyJobID(ctx, tx, recorder, jobID)
	if err != nil {
		return 0, err
	}

	// Get the version bucket
	versionBkt, err := NewBucketPath(BucketJobs, jobID, BucketJobVersions).Get(tx, false)
	if err != nil {
		return 0, NewBoltDBError(err)
	}

	// Get the latest version key
	c := versionBkt.Cursor()
	latestKey, _ := c.Last()
	if latestKey == nil {
		return 0, jobstore.NewErrJobNotFound(jobID)
	}

	version = bytesToUint64(latestKey)
	recorder.Latency(ctx, jobstore.OperationPartDuration, jobstore.AttrOperationPartRead)

	return version, nil
}

// GetJobByIDOrName retrieves a Job identified by either its name or ID.
func (b *BoltJobStore) GetJobByIDOrName(ctx context.Context, idOrName, namespace string) (job models.Job, err error) {
	// First try to get by name
	job, err = b.GetJobByName(ctx, idOrName, namespace)
	if err != nil {
		job, err = b.GetJob(ctx, idOrName)
	}
	return job, err
}
