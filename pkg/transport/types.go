package transport

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
)

// SubscribeFn is provided by an in-process listener as an event callback.
type SubscribeFn func(context.Context, executor.JobEvent, executor.Job)

// Transport is an interface representing a communication channel between
// nodes, through which they can submit, bid on and complete jobs.
type Transport interface {
	/////////////////////////////////////////////////////////////
	/// LIFECYCLE
	/////////////////////////////////////////////////////////////

	// Start the job scheduler. Not that this is blocking and can be managed
	// via the context parameter. You must call Subscribe _before_ starting.
	Start(ctx context.Context) error

	// Shuts down the transport layer and performs resource cleanup.
	Shutdown(ctx context.Context) error

	// HostID returns a unique string per host in whatever network the
	// scheduler is connecting to. Must be unique per instance.
	HostID(ctx context.Context) (string, error)

	/////////////////////////////////////////////////////////////
	/// READ OPERATIONS
	/////////////////////////////////////////////////////////////

	// List returns a list of of known jobs (smart contract will have
	// persistence; libp2p will be lossy).  JobSpec contains everything
	// to do with a job including state, results.
	List(ctx context.Context) (ListResponse, error)

	// Get returns information about the given job.
	Get(ctx context.Context, jobID string) (executor.Job, error)

	// Subscribe registers a callback for updates about any change to a job
	// or its results.  This is in-memory, global, singleton and scoped to the
	// lifetime of the process so no need for an unsubscribe right now.
	Subscribe(ctx context.Context, fn SubscribeFn)

	/////////////////////////////////////////////////////////////
	/// WRITE OPERATIONS - "CLIENT" / REQUESTER NODE
	/////////////////////////////////////////////////////////////

	// Executed by the client (Connie) requesting the work, puts the job into a
	// mempool of work that is available to be done.
	SubmitJob(ctx context.Context, spec executor.JobSpec,
		deal executor.JobDeal) (executor.Job, error)

	// Update the job deal - for example updating concurrency
	UpdateDeal(ctx context.Context, jobID string, deal executor.JobDeal) error

	// Client has decided they no longer want the work done. Can only happen
	// when no runs of the job are in progress.
	CancelJob(ctx context.Context, jobID string) error

	// Executed by the client (Connie) to tell Prue they are good to start.
	// Enables coordination to avoid excess job starting, also allows client to
	// be selective about reputation.
	AcceptJobBid(ctx context.Context, jobID, hostID string) error

	// Executed by the client (Connie) to tell Prue they shouldn't try to run
	// this job.
	RejectJobBid(ctx context.Context, jobID, hostID, message string) error

	/////////////////////////////////////////////////////////////
	/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
	/////////////////////////////////////////////////////////////

	// Executed by the compute node (Prue) when they want to start working on a
	// job.
	BidJob(ctx context.Context, jobID string) error

	// Executed by the compute node when they have completed a job.
	SubmitResult(ctx context.Context, jobID, status, resultsID string) error

	// something has gone wrong with running the job
	// called by the compute node and so will have the nodeID auto-filled
	ErrorJob(ctx context.Context, jobID, status string) error

	// something has gone wrong is checking the job from the requester node
	// called by the requester node and so we need to be given the nodeID.
	ErrorJobForNode(ctx context.Context, jobID, nodeID, status string) error
}

// the data structure a client can use to render a view of the state of the world
// e.g. this is used to render the CLI table and results list
type ListResponse struct {
	Jobs map[string]executor.Job
}

// data structure for a Version response
type VersionResponse struct {
	VersionInfo executor.VersionInfo
}
