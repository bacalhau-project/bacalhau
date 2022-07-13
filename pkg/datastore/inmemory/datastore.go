package inmemory

import (
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/datastore"
	"github.com/filecoin-project/bacalhau/pkg/executor"
)

type InMemoryDatastore struct {
	// we keep pointers to these things because we will update them partially
	jobs          map[string]*executor.Job
	localMetadata map[string]*executor.JobLocalMetadata
	// we don't keep pointers to events because they are immutable
	events map[string][]executor.JobEvent
	mtx    sync.Mutex
}

func NewInMemoryDatastore() (*InMemoryDatastore, error) {
	res := &InMemoryDatastore{
		jobs:          map[string]*executor.Job{},
		localMetadata: map[string]*executor.JobLocalMetadata{},
		events:        map[string][]executor.JobEvent{},
	}
	return res, nil
}

func (d *InMemoryDatastore) GetJob(id string) (datastore.Job, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	job, ok := d.jobs[id]
	if !ok {
		return datastore.Job{}, fmt.Errorf("no job found: %s", id)
	}
	events, ok := d.events[id]
	if !ok {
		events = []executor.JobEvent{}
	}
	localMetadata, ok := d.localMetadata[id]
	if !ok {
		localMetadata = &executor.JobLocalMetadata{}
	}
	return datastore.Job{
		ID:            job.ID,
		Job:           *job,
		LocalMetadata: *localMetadata,
		Events:        events,
	}, nil
}

func (d *InMemoryDatastore) GetJobs(query datastore.ListQuery) ([]datastore.Job, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	result := []datastore.Job{}
	if query.ID != "" {
		job, err := d.GetJob(query.ID)
		if err != nil {
			return result, err
		}
		result = append(result, job)
	}
	return result, nil
}

func (d *InMemoryDatastore) AddJob(job executor.Job) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.jobs[job.ID] = &job
	return nil
}

func (d *InMemoryDatastore) AddEvent(jobID string, ev executor.JobEvent) error {
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

func (d *InMemoryDatastore) UpdateJobDeal(jobID string, deal executor.JobDeal) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	job, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	// TODO: when we remove pointers, we can remove this
	job.Deal = &deal
	return nil
}

func (d *InMemoryDatastore) UpdateJobState(jobID, nodeID string, state executor.JobState) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	job, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	// TODO: when we remove pointers, we can remove this
	job.State[nodeID] = &state
	return nil
}

func (d *InMemoryDatastore) UpdateLocalMetadata(jobID string, data executor.JobLocalMetadata) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	_, ok := d.jobs[jobID]
	if !ok {
		return fmt.Errorf("no job found: %s", jobID)
	}
	d.localMetadata[jobID] = &data
	return nil
}

// Static check to ensure that Transport implements Transport:
var _ datastore.DataStore = (*InMemoryDatastore)(nil)
