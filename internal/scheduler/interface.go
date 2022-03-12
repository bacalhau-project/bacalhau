package scheduler

import "github.com/filecoin-project/bacalhau/internal/types"

type Scheduler interface {

	/////////////////////////////////////////////////////////////
	/// LIFECYCLE
	/////////////////////////////////////////////////////////////

	// Start the scheduler. You must call Subscribe _before_ starting.
	Start() error
	// A unique string per host in whatever network the scheduler is connecting
	// to. Must be unique per instance.
	HostId() (string, error)

	/////////////////////////////////////////////////////////////
	/// READ OPERATIONS
	/////////////////////////////////////////////////////////////

	// List of known jobs (smart contract will have persistence; libp2p will be
	// lossy).  JobSpec contains everything to do with a job including state,
	// results.
	List() (types.ListResponse, error)
	// get a single job
	Get(id string) (*types.Job, error)
	// Listen for updates (subscribe with a callback) about any change to a job
	// or its results.  This is in-memory, global, singleton and scoped to the
	// lifetime of the process so no need for an unsubscribe right now.
	Subscribe(func(jobEvent *types.JobEvent, job *types.Job))

	/////////////////////////////////////////////////////////////
	/// WRITE OPERATIONS - "CLIENT" / REQUESTER
	/////////////////////////////////////////////////////////////

	// Executed by the client (Connie) requesting the work, puts the job into a
	// mempool of work that is available to be done.
	SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error)
	// Update the job deal - for example updating concurrency
	UpdateDeal(jobId string, deal *types.JobDeal) error
	// Client has decided they no longer want the work done. Can only happen
	// when no runs of the job are in progress.
	CancelJob(jobId string) error
	// Executed by the client (Connie) to tell Prue they are good to start.
	// Enables coordination to avoid excess job starting, also allows client to
	// be selective about reputation.
	AcceptJobBid(jobId, hostId string) error
	// Executed by the client (Connie) to tell Prue they shouldn't try to run
	// this job.
	RejectJobBid(jobId, hostId, message string) error
	// Executed by the client when they are satisfied with the outcome of a job
	// (e.g they have completed some verification of a job). Along with the id
	// of the server who did the work this is Input to the reputation system.
	AcceptResult(jobId, hostId string) error
	// Executed by the client when they believe a job has been executed
	// incorrectly. Also input to reputation system.
	RejectResult(jobId, hostId, message string) error

	/////////////////////////////////////////////////////////////
	/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
	/////////////////////////////////////////////////////////////

	// Executed by the compute node (Prue) when they want to start working on a
	// job.
	BidJob(jobId string) error

	// Executed by the compute node when they have completed a job.
	SubmitResult(jobId, status string, results []types.JobStorage) error

	// something has gone wrong with running the job
	// called by the compute node and so will have the nodeId auto-filled
	ErrorJob(jobId, status string) error

	// something has gone wrong is checking the job from the requester node
	// called by the requester node and so we need to be given the nodeId
	ErrorJobForNode(jobId, nodeId, status string) error
}
