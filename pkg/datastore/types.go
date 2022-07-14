package datastore

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
)

type Job struct {
	ID            string
	Data          executor.Job
	LocalMetadata executor.JobLocalMetadata
	Events        []executor.JobEvent
}

type JobQuery struct {
	ID string `json:"id"`
}

// A Datastore will persist jobs and their state to the underlying storage.
// It also gives an efficiernt way to retrieve jobs using queries.
// The Datastore is the local view of the world and the transport
// will get events to other nodes that will update their datastore.
//
// The Datastore and Transport interfaces could be swapped out for some kind
// of smart contract implementation (e.g. FVM)
type DataStore interface {
	GetJob(ctx context.Context, id string) (Job, error)
	GetJobs(ctx context.Context, query JobQuery) ([]Job, error)
	AddJob(ctx context.Context, job executor.Job) error
	AddEvent(ctx context.Context, jobID string, event executor.JobEvent) error
	UpdateJobDeal(ctx context.Context, jobID string, deal executor.JobDeal) error
	UpdateJobState(ctx context.Context, jobID, nodeID string, state executor.JobState) error
	UpdateLocalMetadata(ctx context.Context, jobID string, data executor.JobLocalMetadata) error
}
