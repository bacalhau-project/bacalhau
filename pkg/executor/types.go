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
	//    tells us if the storage resource is "close" i.e. cheap to access
	HasStorageLocally(context.Context, storage.StorageSpec) (bool, error)
	//    tells us how much storage the given volume would consume
	//    which we then use to calculate if there is capacity
	//    alongside cpu & memory usage
	GetVolumeSize(context.Context, storage.StorageSpec) (uint64, error)

	// run the given job - it's expected that we have already prepared the job
	// this will return a local filesystem path to the jobs results
	RunJob(context.Context, Job) (string, error)
}

// Job contains data about a job in the bacalhau network.
type Job struct {
	// The unique global ID of this job in the bacalhau network.
	ID string `json:"id"`

	// The ID of the requester node that owns this job.
	Owner string `json:"owner"`

	// The ID of the requester node that owns this job.
	RequesterNodeID string `json:"requester_node_id"`

	// The ID of the client that created this job.
	ClientID string `json:"client_id"`

	// The specification of this job.
	Spec JobSpec `json:"spec"`

	// The deal the client has made, such as which job bids they have accepted.
	Deal JobDeal `json:"deal"`

	// The states of the job on different compute nodes indexed by node ID.
	State map[string]JobState `json:"state"`

	// Time the job was submitted to the bacalhau network.
	CreatedAt time.Time `json:"created_at"`
}

// JobSpec is a complete specification of a job that can be run on some
// execution provider.
type JobSpec struct {
	// e.g. docker or language
	Engine EngineType `json:"engine"`

	// e.g. ipfs or localfs
	// these verifiers both just copy the results
	// and don't do any verification
	Verifier verifier.VerifierType `json:"verifier"`

	// executor specific data
	Docker   JobSpecDocker   `json:"job_spec_docker"`
	Language JobSpecLanguage `json:"job_spec_language"`

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
type JobSpecDocker struct {
	// this should be pullable by docker
	Image string `json:"image"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"entrypoint"`
	// a map of env to run the container with
	Env []string `json:"env"`
}

// for language style executors (can target docker or wasm)
type JobSpecLanguage struct {
	Language        string `json:"language"`         // e.g. python
	LanguageVersion string `json:"language_version"` // e.g. 3.8
	// must this job be run in a deterministic context?
	Deterministic bool `json:"deterministic"`
	// context is a tar file stored in ipfs, containing e.g. source code and requirements
	Context storage.StorageSpec `json:"context"`
	// optional program specified on commandline, like python -c "print(1+1)"
	Command string `json:"command"`
	// optional program path relative to the context dir. one of Command or ProgramPath must be specified
	ProgramPath string `json:"program_path"`
	// optional requirements.txt (or equivalent) path relative to the context dir
	RequirementsPath string `json:"requirements_path"`
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

// gives us a way to keep local data against a job
// so our compute node and requester node control loops
// can keep state against a job without broadcasting it
// to the rest of the network
type JobLocalMetadata struct {
	ComputeNodeSelected bool `json:"selected"`
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
	JobSpec JobSpec `json:"job_spec"`
	// this is only defined in "update_deal" events
	JobDeal JobDeal `json:"job_deal"`
	// most other events are a case of a client<->node state change
	JobState  JobState  `json:"job_state"`
	EventTime time.Time `json:"event_time"`
}

// Version of a bacalhau binary (either client or server)
type VersionInfo struct {
	// Client Version: version.Info{Major:"1", Minor:"24", GitVersion:"v1.24.0",
	// GitCommit:"4ce5a8954017644c5420bae81d72b09b735c21f0", GitTreeState:"clean",
	// BuildDate:"2022-05-03T13:46:05Z", GoVersion:"go1.18.1", Compiler:"gc", Platform:"darwin/arm64"}

	Major      string    `json:"major,omitempty" yaml:"major,omitempty"`
	Minor      string    `json:"minor,omitempty" yaml:"minor,omitempty"`
	GitVersion string    `json:"gitversion" yaml:"gitversion"`
	GitCommit  string    `json:"gitcommit" yaml:"gitcommit"`
	BuildDate  time.Time `json:"builddate" yaml:"builddate"`
	GOOS       string    `json:"goos" yaml:"goos"`
	GOARCH     string    `json:"goarch" yaml:"goarch"`
}
