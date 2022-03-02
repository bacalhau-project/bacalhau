package types

import "github.com/filecoin-project/bacalhau/internal/types"

type Scheduler interface {
	/// READ OPERATIONS
	// List of known jobs (smart contract will have persistence; libp2p will be
	// lossy).  JobSpec contains everything to do with a job including state,
	// results.
	List() (types.ListResponse, error)
	// Listen for updates (subscribe with a callback) about any change to a job
	// or its results.  This is in-memory, global and scoped to the lifetime of
	// the process so no need for an unsubscribe right now
	Subscribe(func(eventName string, job *types.JobData))

	/// WRITE OPERATIONS - "CLIENT" / REQUESTER
	// Executed by the client (Connie) requesting the work, puts the job into a
	// mempool of work that is available to be done.
	SubmitJob(spec *types.JobSpec) error
	// Client has decided they no longer want the work done. Can only happen
	// when no runs of the job are in progress.
	CancelJob(jobId string) error
	// Executed by the client (Connie) to tell Prue they are good to start.
	// Enables coordination to avoid excess job starting, also allows client to
	// be selective about reputation.
	ApproveJobBid(jobId string) error
	// Executed by the client (Connie) to tell Prue they shouldn't try to run
	// this job.
	RejectJobBid(jobId string) error
	// Update a named field of a job field for example updating concurrency
	UpdateJob(jobId, field, value string) error
	// Executed by the client when they are satisfied with the outcome of a job
	// (e.g they have completed some verification of a job). Along with the id
	// of the server who did the work this is Input to the reputation system.
	ApproveResult(jobId, resultId string) error
	// Executed by the client when they believe a job has been executed
	// incorrectly. Also input to reputation system.
	RejectResult(jobId, resultId string) error

	/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
	// Executed by the compute node (Prue) when they want to start working on a
	// job. Returns resultId.
	BidJob(jobId string) (string, error)
	// Executed by the compute node when they have partially or fully completed
	// a job and have some results (and possibly evidence of computation).
	// Optionally includes a nullable result pointer which points to where the
	// results are written to a storage implementation (e.g. IPFS).
	SubmitProgress(jobId, resultId, state, status string, resultPointer *string) error
}
