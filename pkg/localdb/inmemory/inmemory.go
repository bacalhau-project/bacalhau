package inmemory

import (
	"context"
	"fmt"
	"time"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
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

func (d *InMemoryDatastore) GetJob(ctx context.Context, id string) (model.Job, error) {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.GetJob")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	job, ok := d.jobs[id]
	if !ok {
		return model.Job{}, fmt.Errorf("no job found: %s", id)
	}
	return *job, nil
}

func (d *InMemoryDatastore) GetJobEvents(ctx context.Context, id string) ([]model.JobEvent, error) {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.GetJobEvents")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[id]
	if !ok {
		return []model.JobEvent{}, fmt.Errorf("no job found: %s", id)
	}
	result, ok := d.events[id]
	if !ok {
		result = []model.JobEvent{}
	}
	return result, nil
}

func (d *InMemoryDatastore) GetJobLocalEvents(ctx context.Context, id string) ([]model.JobLocalEvent, error) {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.GetJobLocalEvents")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[id]
	if !ok {
		return []model.JobLocalEvent{}, fmt.Errorf("no job found: %s", id)
	}
	result, ok := d.localEvents[id]
	if !ok {
		result = []model.JobLocalEvent{}
	}
	return result, nil
}

func (d *InMemoryDatastore) GetJobs(ctx context.Context, query localdb.JobQuery) ([]model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.GetJobs")
	defer span.End()

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	result := []model.Job{}

	if query.ID != "" {
		job, err := d.GetJob(ctx, query.ID)
		if err != nil {
			return result, err
		}
		result = append(result, job)
	} else {
		for _, job := range d.jobs {
			result = append(result, *job)
		}
	}
	return result, nil
}

func (d *InMemoryDatastore) AddJob(ctx context.Context, job model.Job) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.AddJob")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	existingJob, ok := d.jobs[job.ID]
	if ok {
		if len(job.RequesterPublicKey) > 0 {
			existingJob.RequesterPublicKey = job.RequesterPublicKey
		}
		return nil
	}
	d.jobs[job.ID] = &job
	return nil
}

func (d *InMemoryDatastore) AddEvent(ctx context.Context, jobID string, ev model.JobEvent) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.AddEvent")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
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
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.AddLocalEvent")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	eventArr, ok := d.localEvents[jobID]
	if !ok {
		eventArr = []model.JobLocalEvent{}
	}
	eventArr = append(eventArr, ev)
	d.localEvents[jobID] = eventArr
	return nil
}

func (d *InMemoryDatastore) UpdateJobDeal(ctx context.Context, jobID string, deal model.JobDeal) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.UpdateJobDeal")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	job, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	job.Deal = deal
	return nil
}

func (d *InMemoryDatastore) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.GetJobState")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return model.JobState{}, fmt.Errorf("no job found: %s", jobID)
	}
	state, ok := d.states[jobID]
	if !ok {
		return model.JobState{}, nil
	}
	// copy job state because it has mutable fields (Nodes), we should return a
	// value that isn't concurrently being modified
	// XXX what about the mutable fields within JobNodeState :-(
	newJobState := model.JobState{
		Nodes: map[string]model.JobNodeState{},
	}
	for idx, node := range state.Nodes {
		newJobState.Nodes[idx] = node
	}
	return newJobState, nil
}

func (d *InMemoryDatastore) UpdateShardState(
	ctx context.Context,
	jobID, nodeID string,
	shardIndex int,
	update model.JobShardState,
) error {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/localdb/inmemory/InMemoryDatastore.UpdateShardState")
	defer span.End()

	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	jobState, ok := d.states[jobID]
	if !ok {
		jobState = &model.JobState{
			Nodes: map[string]model.JobNodeState{},
		}
	}
	nodeState, ok := jobState.Nodes[nodeID]
	if !ok {
		nodeState = model.JobNodeState{
			Shards: map[int]model.JobShardState{},
		}
	}
	shardSate, ok := nodeState.Shards[shardIndex]
	if !ok {
		shardSate = model.JobShardState{
			NodeID:     nodeID,
			ShardIndex: shardIndex,
		}
	}

	shardSate.State = update.State
	if update.Status != "" {
		shardSate.Status = update.Status
	}

	if update.RunOutput != nil {
		shardSate.RunOutput = update.RunOutput
	}

	if len(update.VerificationProposal) != 0 {
		shardSate.VerificationProposal = update.VerificationProposal
	}

	if update.VerificationResult.Complete {
		shardSate.VerificationResult = update.VerificationResult
	}

	if model.IsValidStorageSourceType(update.PublishedResult.Engine) {
		shardSate.PublishedResult = update.PublishedResult
	}

	nodeState.Shards[shardIndex] = shardSate
	jobState.Nodes[nodeID] = nodeState
	d.states[jobID] = jobState
	return nil
}

// Static check to ensure that Transport implements Transport:
var _ localdb.LocalDB = (*InMemoryDatastore)(nil)
