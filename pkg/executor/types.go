package executor

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
)

type APIVersion string

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
	RunShard(
		ctx context.Context,
		job Job,
		shardIndex int,
		resultsDir string,
	) error
}

// Job contains data about a job in the bacalhau network.
type Job struct {
	// The unique global ID of this job in the bacalhau network.
	ID string `json:"id"`

	// The ID of the requester node that owns this job.
	RequesterNodeID string `json:"requester_node_id"`

	// The ID of the client that created this job.
	ClientID string `json:"client_id"`

	// The specification of this job.
	Spec JobSpec `json:"spec"`

	// The deal the client has made, such as which job bids they have accepted.
	Deal JobDeal `json:"deal"`

	// how will this job be executed by nodes on the network
	ExecutionPlan JobExecutionPlan `json:"execution_plan"`

	// Time the job was submitted to the bacalhau network.
	CreatedAt time.Time `json:"created_at"`
}

type JobExecutionPlan struct {
	// how many shards are there in total for this job
	// we are expecting this number x concurrency total
	// JobShardState objects for this job
	TotalShards int `json:"shards_total"`
}

// describe how we chunk a job up into shards
type JobShardingConfig struct {
	// divide the inputs up into the smallest possible unit
	// for example /* would mean "all top level files or folders"
	// this being an empty string means "no sharding"
	GlobPattern string `json:"glob_pattern" yaml:"glob_pattern"`
	// how many "items" are to be processed in each shard
	// we first apply the glob pattern which will result in a flat list of items
	// this number decides how to group that flat list into actual shards run by compute nodes
	BatchSize int `json:"batch_size" yaml:"batch_size"`
	// when using multiple input volumes
	// what path do we treat as the common mount path to apply the glob pattern to
	BasePath string `json:"glob_pattern_base_path" yaml:"glob_pattern_base_path"`
}

// The state of a job across the whole network
// generally be in different states on different nodes - one node may be
// ignoring a job as its bid was rejected, while another node may be
// submitting results for the job to the requester node.
// Each node will produce an array of JobShardState one for each shard
// (jobs without a sharding config will still have sharded job
// states - just with a shard count of 1). Any code that is determining
// the current "state" of a job must look at both
// the ShardCount of the JobExecutionPlan and the
// collection of JobShardState to determine the current state.

// JobState itself is not mutable - the JobExecutionPlan and
// JobShardState are updatable and the JobState is queried by the rest
// of the system
type JobState struct {
	Nodes map[string]JobNodeState `json:"nodes"`
}

type JobNodeState struct {
	Shards map[int]JobShardState `json:"shards"`
}

type JobShardState struct {
	// which node is running this shard
	NodeID string `json:"node_id"`
	// what shard is this we are running
	ShardIndex int `json:"shard_index"`
	// what is the state of the shard on this node
	State JobStateType `json:"state"`
	// an arbitrary status message
	Status string `json:"status"`
	// the ID of the results for this shard
	// this will be resolved by the verifier somehow
	ResultsID string `json:"results_id"`
}

// The deal the client has made with the bacalhau network.
// This is updateable by the client who submitted the job
type JobDeal struct {
	// The maximum number of concurrent compute node bids that will be
	// accepted by the requester node on behalf of the client.
	Concurrency int `json:"concurrency"`
}

// JobSpec is a complete specification of a job that can be run on some
// execution provider.
type JobSpec struct {
	APIVersion APIVersion `json:"apiVersion" yaml:"apiVersion"`
	// e.g. docker or language
	Engine EngineType `json:"engine,omitempty" yaml:"engine,omitempty"`
	// allow the engine to be provided as a string for yaml and JSON job specs
	EngineName string `json:"engine_name" yaml:"engine_name"`

	// e.g. ipfs or localfs
	// these verifiers both just copy the results
	// and don't do any verification
	Verifier verifier.VerifierType `json:"verifier" yaml:"verifier"`
	// allow the verifier to be provided as a string for yaml and JSON job specs
	VerifierName string `json:"verifier_name" yaml:"verifier_name"`

	// executor specific data
	Docker   JobSpecDocker   `json:"job_spec_docker,omitempty" yaml:"job_spec_docker,omitempty"`
	Language JobSpecLanguage `json:"job_spec_language,omitempty" yaml:"job_spec_language,omitempty"`

	// the compute (cpy, ram) resources this job requires
	Resources capacitymanager.ResourceUsageConfig `json:"resources" yaml:"resources"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	Inputs []storage.StorageSpec `json:"inputs" yaml:"inputs"`

	// Input volumes that will not be sharded
	// for example to upload code into a base image
	// every shard will get the full range of context volumes
	Contexts []storage.StorageSpec `json:"contexts" yaml:"contexts"`

	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []storage.StorageSpec `json:"outputs" yaml:"outputs"`

	// Annotations on the job - could be user or machine assigned
	Annotations []string `json:"annotations" yaml:"annotations"`

	// the sharding config for this job
	// describes how the job might be split up into parallel shards
	Sharding JobShardingConfig `json:"sharding" yaml:"sharding"`
}

// for VM style executors
type JobSpecDocker struct {
	// this should be pullable by docker
	Image string `json:"image" yaml:"image"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"entrypoint" yaml:"entrypoint"`
	// a map of env to run the container with
	Env []string `json:"env" yaml:"env"`
	// working directory inside the container
	WorkingDir string `json:"workdir" yaml:"workdir"`
}

// for language style executors (can target docker or wasm)
type JobSpecLanguage struct {
	Language        string `json:"language" yaml:"language"`                 // e.g. python
	LanguageVersion string `json:"language_version" yaml:"language_version"` // e.g. 3.8
	// must this job be run in a deterministic context?
	Deterministic bool `json:"deterministic" yaml:"deterministic"`
	// context is a tar file stored in ipfs, containing e.g. source code and requirements
	Context storage.StorageSpec `json:"context" yaml:"context"`
	// optional program specified on commandline, like python -c "print(1+1)"
	Command string `json:"command" yaml:"command"`
	// optional program path relative to the context dir. one of Command or ProgramPath must be specified
	ProgramPath string `json:"program_path" yaml:"program_path"`
	// optional requirements.txt (or equivalent) path relative to the context dir
	RequirementsPath string `json:"requirements_path" yaml:"requirements_path"`
}

// gives us a way to keep local data against a job
// so our compute node and requester node control loops
// can keep state against a job without broadcasting it
// to the rest of the network
type JobLocalEvent struct {
	EventName    JobLocalEventType `json:"event_name"`
	JobID        string            `json:"job_id"`
	ShardIndex   int               `json:"shard_index"`
	TargetNodeID string            `json:"target_node_id"`
}

// we emit these to other nodes so they update their
// state locally and can emit events locally
type JobEvent struct {
	JobID string `json:"job_id"`
	// what shard is this event for
	ShardIndex int `json:"shard_index"`
	// optional clientID if this is an externally triggered event (like create job)
	ClientID string `json:"client_id"`
	// the node that emitted this event
	SourceNodeID string `json:"source_node_id"`
	// the node that this event is for
	// e.g. "AcceptJobBid" was emitted by requestor but it targeting compute node
	TargetNodeID string       `json:"target_node_id"`
	EventName    JobEventType `json:"event_name"`
	// this is only defined in "create" events
	JobSpec JobSpec `json:"job_spec"`
	// this is only defined in "create" events
	JobExecutionPlan JobExecutionPlan `json:"job_execution_plan"`
	// this is only defined in "update_deal" events
	JobDeal   JobDeal   `json:"job_deal"`
	Status    string    `json:"status"`
	ResultsID string    `json:"results_id"`
	EventTime time.Time `json:"event_time"`
}

type JobCreatePayload struct {
	// the id of the client that is submitting the job
	ClientID string `json:"client_id"`

	// The job specification:
	Spec JobSpec `json:"spec"`

	// The deal the client has made with the network, at minimum this should
	// contain the client's ID for verifying the message authenticity:
	Deal JobDeal `json:"deal"`

	// Optional base64-encoded tar file that will be pinned to IPFS and
	// mounted as storage for the job. Not part of the spec so we don't
	// flood the transport layer with it (potentially very large).
	Context string `json:"context,omitempty"`
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
