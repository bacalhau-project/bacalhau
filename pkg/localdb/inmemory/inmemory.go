package inmemory

import (
	"context"
	"sort"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"
	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type InMemoryDatastore struct {
	// we keep pointers to these things because we will update them partially
	jobs        map[string]*model.Job
	states      map[string]*model.JobState
	events      map[string][]model.JobEvent
	localEvents map[string][]model.JobLocalEvent
	mtx         sync.RWMutex
}

func NewInMemoryDatastore() (*InMemoryDatastore, error) {
	res := &InMemoryDatastore{
		jobs:        map[string]*model.Job{},
		states:      map[string]*model.JobState{},
		events:      map[string][]model.JobEvent{},
		localEvents: map[string][]model.JobLocalEvent{},
	}
	res.mtx.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InMemoryDatastore.mtx",
	})
	return res, nil
}

// Gets a job from the datastore.
//
// Errors:
//
//   - error-job-not-found        		  -- if the job is not found
func (d *InMemoryDatastore) GetJob(ctx context.Context, id string) (*model.Job, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.GetJob")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getJob(id)
}

// Get Job Events from a job ID
//
// Errors:
//
//   - error-job-not-found        		  -- if the job is not found
func (d *InMemoryDatastore) GetJobEvents(ctx context.Context, id string) ([]model.JobEvent, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.GetJobEvents")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[id]
	if !ok {
		return []model.JobEvent{}, bacerrors.NewJobNotFound(id)
	}
	result, ok := d.events[id]
	if !ok {
		result = []model.JobEvent{}
	}
	return result, nil
}

func (d *InMemoryDatastore) GetJobLocalEvents(ctx context.Context, id string) ([]model.JobLocalEvent, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.GetJobLocalEvents")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[id]
	if !ok {
		return []model.JobLocalEvent{}, bacerrors.NewJobNotFound(id)
	}
	result, ok := d.localEvents[id]
	if !ok {
		result = []model.JobLocalEvent{}
	}
	return result, nil
}

func (d *InMemoryDatastore) GetJobs(ctx context.Context, query localdb.JobQuery) ([]*model.Job, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.GetJobs")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	result := []*model.Job{}

	if query.ID != "" {
		log.Ctx(ctx).Trace().Msgf("querying for single job %s", query.ID)
		j, err := d.getJob(query.ID)
		if err != nil {
			return nil, err
		}
		return []*model.Job{j}, nil
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

func (d *InMemoryDatastore) GetJobsCount(ctx context.Context, query localdb.JobQuery) (int, error) {
	useQuery := query
	useQuery.Limit = 0
	useQuery.Offset = 0
	jobs, err := d.GetJobs(ctx, useQuery)
	if err != nil {
		return 0, err
	}
	return len(jobs), nil
}

func (d *InMemoryDatastore) HasLocalEvent(ctx context.Context, jobID string, eventFilter localdb.LocalEventFilter) (bool, error) {
	jobLocalEvents, err := d.GetJobLocalEvents(ctx, jobID)
	if err != nil {
		return false, err
	}
	hasEvent := false
	for _, localEvent := range jobLocalEvents {
		if eventFilter(localEvent) {
			hasEvent = true
			break
		}
	}
	return hasEvent, nil
}

func (d *InMemoryDatastore) AddJob(ctx context.Context, j *model.Job) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.AddJob")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	existingJob, ok := d.jobs[j.Metadata.ID]
	if ok {
		if len(j.Status.Requester.RequesterPublicKey) > 0 {
			existingJob.Status.Requester.RequesterPublicKey = j.Status.Requester.RequesterPublicKey
		}
		return nil
	}
	d.jobs[j.Metadata.ID] = j
	return nil
}

func (d *InMemoryDatastore) AddEvent(ctx context.Context, jobID string, ev model.JobEvent) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.AddEvent")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return bacerrors.NewJobNotFound(jobID)
	}
	eventArr, ok := d.events[jobID]
	if !ok {
		eventArr = []model.JobEvent{}
	}
	eventArr = append(eventArr, ev)
	d.events[jobID] = eventArr
	return nil
}

func (d *InMemoryDatastore) AddLocalEvent(ctx context.Context, jobID string, ev model.JobLocalEvent) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.AddLocalEvent")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return bacerrors.NewJobNotFound(jobID)
	}
	eventArr, ok := d.localEvents[jobID]
	if !ok {
		eventArr = []model.JobLocalEvent{}
	}
	eventArr = append(eventArr, ev)
	d.localEvents[jobID] = eventArr
	return nil
}

func (d *InMemoryDatastore) UpdateJobDeal(ctx context.Context, jobID string, deal model.Deal) error {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.UpdateJobDeal")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	job, ok := d.jobs[jobID]
	if !ok {
		return bacerrors.NewJobNotFound(jobID)
	}
	job.Spec.Deal = deal
	return nil
}

func (d *InMemoryDatastore) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.GetJobState")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return model.JobState{}, bacerrors.NewJobNotFound(jobID)
	}
	state, ok := d.states[jobID]
	if !ok {
		return model.JobState{}, nil
	}
	// return a copy so we remain within the mutex of the localdb
	// in terms of accessing d.states
	return *state, nil
}

func (d *InMemoryDatastore) UpdateShardState(
	ctx context.Context,
	jobID, nodeID string,
	shardIndex int,
	update model.JobShardState,
) error {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/localdb/inmemory.InMemoryDatastore.UpdateShardState")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return bacerrors.NewJobNotFound(jobID)
	}
	jobState, ok := d.states[jobID]
	if !ok {
		jobState = &model.JobState{
			Nodes: map[string]model.JobNodeState{},
		}
	}
	err := shared.UpdateShardState(nodeID, shardIndex, jobState, update)
	if err != nil {
		return err
	}
	d.states[jobID] = jobState
	return nil
}

// helper method to read a single job from memory. This is used by both GetJob and GetJobs.
// It is important that we don't attempt to acquire a lock inside this method to avoid deadlocks since
// the callers are expected to be holding a lock, and golang doesn't support reentrant locks.
func (d *InMemoryDatastore) getJob(id string) (*model.Job, error) {
	if len(id) < model.ShortIDLength {
		return nil, bacerrors.NewJobNotFound(id)
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
		return nil, returnError
	}

	return j, nil
}

// Static check to ensure that Transport implements Transport:
var _ localdb.LocalDB = (*InMemoryDatastore)(nil)
