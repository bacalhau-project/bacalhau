package executor

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/resourceusage"
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

// Job contains data about a job in the bacalhau network.
type Job struct {
	// The unique global ID of this job in the bacalhau network.
	ID string `json:"id"`

	// The ID of the requester node that owns this job.
	Owner string `json:"owner"`

	// The specification of this job.
	Spec *JobSpec `json:"spec"`

	// The deal the client has made, such as which job bids they have accepted.
	Deal *JobDeal `json:"deal"`

	// The states of the job on different compute nodes indexed by node ID.
	State map[string]*JobState `json:"state"`

	// Time the job was submitted to the bacalhau network.
	CreatedAt time.Time `json:"created_at"`
}

// Copy returns a deep copy of the given job.
// TODO: use a library for deepcopies, this is tedious and likely to be
//       fragile to changes in the Job type.
func (j *Job) Copy() Job {
	jc := *j
	jc.State = make(map[string]*JobState)
	for k, v := range j.State {
		jc.State[k] = v
	}
	if j.Spec != nil {
		sc := *j.Spec
		jc.Spec = &sc
	}
	if j.Deal != nil {
		dc := *j.Deal
		jc.Deal = &dc
	}

	return jc
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

	// the compute (cpy, ram) resources this job requires
	Resources resourceusage.ResourceUsageConfig `json:"resources"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	Inputs []storage.StorageSpec `json:"inputs"`
	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []storage.StorageSpec `json:"outputs"`

	// Annotations on the job - could be user or machine assigned
	Annotations []string
}

// for VM style executors
type JobSpecVM struct {
	// this should be pullable by docker
	Image string `json:"image"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"entrypoint"`
	// a map of env to run the container with
	Env []string `json:"env"`
}

// for Wasm style executors
type JobSpecWasm struct {
	Bytecode storage.StorageSpec `json:"bytecode"`
}

// The state of a job on a particular compute node. Note that the job will
// generally be in different states on different nodes - one node may be
// ignoring a job as its bid was rejected, while another node may be
// submitting results for the job to the requester node.
type JobState struct {
	State     JobStateType `json:"state"`
	Status    string       `json:"status"`
	ResultsID string       `json:"results_id"`
}

// The deal the client has made with the bacalhau network.
type JobDeal struct {
	// The ID of the client that created this job.
	ClientID string `json:"client_id"`

	// The maximum number of concurrent compute node bids that will be
	// accepted by the requester node on behalf of the client.
	Concurrency int `json:"concurrency"`

	// The compute node bids that have been accepted by the requester node on
	// behalf of the client. Nodes that do not have accepted bids may still
	// run and submit results for a job - this could be used to create a
	// reputation system for new compute nodes.
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
