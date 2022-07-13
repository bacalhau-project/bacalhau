package datastore

import "github.com/filecoin-project/bacalhau/pkg/executor"

type Job struct {
	ID            string
	Job           executor.Job
	LocalMetadata executor.JobLocalMetadata
	Events        []executor.JobEvent
}

type ListQuery struct {
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
	GetJob(id string) (Job, error)
	GetJobs(query ListQuery) ([]Job, error)
	AddJob(job executor.Job) error
	AddEvent(jobID string, event executor.JobEvent) error
	UpdateJobDeal(jobID string, deal executor.JobDeal) error
	UpdateJobState(jobID, nodeID string, state executor.JobState) error
	UpdateLocalMetadata(jobID string, data executor.JobLocalMetadata) error
}
