package executor

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
)

// Executor represents an execution provider, which can execute jobs on some
// kind of backend, such as a docker daemon.
type Executor interface {
	// tells you if the required software is installed on this machine
	// this is used in job selection
	IsInstalled(context.Context) (bool, error)

	// used to filter and select jobs
	HasStorage(context.Context, storage.StorageSpec) (bool, error)

	// run the given job - it's expected that we have already prepared the job
	// this will return a local filesystem path to the jobs results
	RunJob(context.Context, *Job) (string, error)
}

// Job contains data about a job running on some execution provider.
type Job struct {
	ID string `json:"id"`
	// the client node that "owns" this job (as in who submitted it)
	Owner string   `json:"owner"`
	Spec  *JobSpec `json:"spec"`
	Deal  *JobDeal `json:"deal"`
	// a map of nodeID -> state of the job on that node
	State     map[string]*JobState `json:"state"`
	CreatedAt time.Time            `json:"created_at"`
}

// JobSpec is a complete specification of a job that can be run on some
// execution provider.
type JobSpec struct {
	// e.g. firecracker, docker or wasm
	Engine EngineType `json:"engine"`

	// e.g. ipfs or localfs
	// these verifiers both just copy the results
	// and don't do any verification
	Verifier verifier.VerifierType `json:"verifier"`

	// for VM based executors
	VM   JobSpecVM   `json:"job_spec_vm"`
	Wasm JobSpecWasm `json:"job_spec_wasm"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	Inputs []storage.StorageSpec `json:"inputs"`
	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []storage.StorageSpec `json:"outputs"`
}

// for VM style executors
type JobSpecVM struct {
	// this should be pullable by docker
	Image string `json:"image"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"entrypoint"`
	// a map of env to run the container with
	Env []string `json:"env"`
	// https://github.com/BTBurke/k8sresource strings
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Disk   string `json:"disk"`
}

// for Wasm style executors
type JobSpecWasm struct {
	Bytecode storage.StorageSpec `json:"bytecode"`
}

// keep track of job states on a particular node
type JobState struct {
	State     JobStateType `json:"state"`
	Status    string       `json:"status"`
	ResultsID string       `json:"results_id"`
}

// omly the client can update this as it's the client that will
// pay out based on the deal
type JobDeal struct {
	// how many nodes do we want to run this job?
	Concurrency int `json:"concurrency"`
	// the nodes we have assigned (and will pay)
	// other nodes are welcome to submit results without having been assigned
	// this is how they can bootstrap their reputation
	AssignedNodes []string `json:"assigned_nodes"`
}

// we emit these to other nodes so they update their
// state locally and can emit events locally
type JobEvent struct {
	JobID     string       `json:"job_id"`
	NodeID    string       `json:"node_id"`
	EventName JobEventType `json:"event_name"`
	// this is only defined in "create" events
	JobSpec *JobSpec `json:"job_spec"`
	// this is only defined in "update_deal" events
	JobDeal *JobDeal `json:"job_deal"`
	// most other events are a case of a client<->node state change
	JobState  *JobState `json:"job_state"`
	EventTime time.Time `json:"event_time"`
}
