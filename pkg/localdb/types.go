package localdb

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
)

type JobQuery struct {
	ID string `json:"id"`
}

// A LocalDB will persist jobs and their state to the underlying storage.
// It also gives an efficiernt way to retrieve jobs using queries.
// The LocalDB is the local view of the world and the transport
// will get events to other nodes that will update their datastore.
//
// The LocalDB and Transport interfaces could be swapped out for some kind
// of smart contract implementation (e.g. FVM)
type LocalDB interface {
	GetJob(ctx context.Context, id string) (executor.Job, error)
	GetJobEvents(ctx context.Context, id string) ([]executor.JobEvent, error)
	GetJobLocalEvents(ctx context.Context, id string) ([]executor.JobLocalEvent, error)
	GetJobs(ctx context.Context, query JobQuery) ([]executor.Job, error)
	AddJob(ctx context.Context, job executor.Job) error
	AddEvent(ctx context.Context, jobID string, event executor.JobEvent) error
	AddLocalEvent(ctx context.Context, jobID string, event executor.JobLocalEvent) error
	UpdateJobDeal(ctx context.Context, jobID string, deal executor.JobDeal) error
	UpdateExecutionState(ctx context.Context, jobID, nodeID string, state executor.JobState) error
}
