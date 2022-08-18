package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
)

type InMemoryDatastore struct {
	// we keep pointers to these things because we will update them partially
	jobs        map[string]*executor.Job
	states      map[string]*executor.JobState
	events      map[string][]executor.JobEvent
	localEvents map[string][]executor.JobLocalEvent
	mtx         sync.RWMutex
}

func NewInMemoryDatastore() (*InMemoryDatastore, error) {
	res := &InMemoryDatastore{
		jobs:        map[string]*executor.Job{},
		states:      map[string]*executor.JobState{},
		events:      map[string][]executor.JobEvent{},
		localEvents: map[string][]executor.JobLocalEvent{},
	}
	return res, nil
}

func (d *InMemoryDatastore) GetJob(ctx context.Context, id string) (executor.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	job, ok := d.jobs[id]
	if !ok {
		return executor.Job{}, fmt.Errorf("no job found: %s", id)
	}
	return *job, nil
}

func (d *InMemoryDatastore) GetJobEvents(ctx context.Context, id string) ([]executor.JobEvent, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[id]
	if !ok {
		return []executor.JobEvent{}, fmt.Errorf("no job found: %s", id)
	}
	result, ok := d.events[id]
	if !ok {
		result = []executor.JobEvent{}
	}
	return result, nil
}

func (d *InMemoryDatastore) GetJobLocalEvents(ctx context.Context, id string) ([]executor.JobLocalEvent, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[id]
	if !ok {
		return []executor.JobLocalEvent{}, fmt.Errorf("no job found: %s", id)
	}
	result, ok := d.localEvents[id]
	if !ok {
		result = []executor.JobLocalEvent{}
	}
	return result, nil
}

func (d *InMemoryDatastore) GetJobs(ctx context.Context, query localdb.JobQuery) ([]executor.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	result := []executor.Job{}
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

func (d *InMemoryDatastore) AddJob(ctx context.Context, job executor.Job) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[job.ID]
	if ok {
		return nil
	}
	d.jobs[job.ID] = &job
	return nil
}

func (d *InMemoryDatastore) AddEvent(ctx context.Context, jobID string, ev executor.JobEvent) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	eventArr, ok := d.events[jobID]
	if !ok {
		eventArr = []executor.JobEvent{}
	}
	eventArr = append(eventArr, ev)
	d.events[jobID] = eventArr
	return nil
}

func (d *InMemoryDatastore) AddLocalEvent(ctx context.Context, jobID string, ev executor.JobLocalEvent) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	eventArr, ok := d.localEvents[jobID]
	if !ok {
		eventArr = []executor.JobLocalEvent{}
	}
	eventArr = append(eventArr, ev)
	d.localEvents[jobID] = eventArr
	return nil
}

func (d *InMemoryDatastore) UpdateJobDeal(ctx context.Context, jobID string, deal executor.JobDeal) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	job, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	job.Deal = deal
	return nil
}

func (d *InMemoryDatastore) GetJobState(ctx context.Context, jobID string) (executor.JobState, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return executor.JobState{}, fmt.Errorf("no job found: %s", jobID)
	}
	state, ok := d.states[jobID]
	if !ok {
		return executor.JobState{}, nil
	}
	// copy job state because it has mutable fields (Nodes), we should return a
	// value that isn't concurrently being modified
	// XXX what about the mutable fields within JobNodeState :-(
	newJobState := executor.JobState{
		Nodes: map[string]executor.JobNodeState{},
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
	update executor.JobShardState,
) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	jobState, ok := d.states[jobID]
	if !ok {
		jobState = &executor.JobState{
			Nodes: map[string]executor.JobNodeState{},
		}
	}
	nodeState, ok := jobState.Nodes[nodeID]
	if !ok {
		nodeState = executor.JobNodeState{
			Shards: map[int]executor.JobShardState{},
		}
	}
	shardSate, ok := nodeState.Shards[shardIndex]
	if !ok {
		shardSate = executor.JobShardState{
			NodeID:     nodeID,
			ShardIndex: shardIndex,
		}
	}

	shardSate.State = update.State
	if update.Status != "" {
		shardSate.Status = update.Status
	}

	if update.ResultsID != "" {
		shardSate.ResultsID = update.ResultsID
	}

	nodeState.Shards[shardIndex] = shardSate
	jobState.Nodes[nodeID] = nodeState
	d.states[jobID] = jobState
	return nil
}

// Static check to ensure that Transport implements Transport:
var _ localdb.LocalDB = (*InMemoryDatastore)(nil)
