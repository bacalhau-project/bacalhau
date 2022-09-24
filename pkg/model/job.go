package model

import (
	"fmt"
	"time"
)

// Job contains data about a job in the bacalhau network.
type Job struct {
	// The unique global ID of this job in the bacalhau network.
	ID string `json:"id"`

	// The ID of the requester node that owns this job.
	RequesterNodeID string `json:"requester_node_id"`

	// The public key of the requestor node that created this job
	// This can be used to encrypt messages back to the creator
	RequesterPublicKey []byte `json:"requester_public_key"`

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

// JobShard contains data about a job shard in the bacalhau network.
type JobShard struct {
	Job Job `json:"job"`

	Index int `json:"index"`
}

func (shard JobShard) ID() string {
	return fmt.Sprintf("%s:%d", shard.Job.ID, shard.Index)
}

func (shard JobShard) String() string {
	return shard.ID()
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
	// the proposed results for this shard
	// this will be resolved by the verifier somehow
	VerificationProposal []byte             `json:"verification_proposal"`
	VerificationResult   VerificationResult `json:"verification_result"`
	PublishedResult      StorageSpec        `json:"published_results"`
	RunOutput            *RunCommandResult  `json:"run_output"`
}

// The deal the client has made with the bacalhau network.
// This is updateable by the client who submitted the job
type JobDeal struct {
	// The maximum number of concurrent compute node bids that will be
	// accepted by the requester node on behalf of the client.
	Concurrency int `json:"concurrency"`
	// The number of nodes that must agree on a verification result
	// this is used by the different verifiers - for example the
	// deterministic verifier requires the winning group size
	// to be at least this size
	Confidence int `json:"confidence"`
	// The minimum number of bids that must be received before the requestor
	// node will randomly accept concurrency-many of them. This allows the
	// requestor node to get some level of guarantee that the execution of the
	// jobs will be spread evenly across the network (assuming that this value
	// is some large proportion of the size of the network).
	MinBids int `json:"min_bids"`
}

// JobSpec is a complete specification of a job that can be run on some
// execution provider.
type JobSpec struct {
	APIVersion APIVersion `json:"apiVersion" yaml:"apiVersion"`
	// e.g. docker or language
	Engine EngineType `json:"engine,omitempty" yaml:"engine,omitempty"`
	// allow the engine to be provided as a string for yaml and JSON job specs
	EngineName string `json:"engine_name" yaml:"engine_name" jsonschema_extras:"enum=noop,enum=docker,enum=wasm,enum=language,enum=pythonwasm"`

	Verifier VerifierType `json:"verifier,omitempty" yaml:"verifier,omitempty"`
	// allow the verifier to be provided as a string for yaml and JSON job specs
	VerifierName string `json:"verifier_name" yaml:"verifier_name" jsonschema_extras:"enum=Noop,enum=Deterministicverifier"`

	// there can be multiple publishers for the job
	Publisher PublisherType `json:"publisher,omitempty" yaml:"publisher,omitempty"`
	//nolint:gochecknoinits
	PublisherName string `json:"publisher_name" yaml:"publisher_name" jsonschema_extras:"enum=Noop,enum=ipfs,enum=filecoin,enum=estuary"`

	// executor specific data
	Docker   JobSpecDocker   `json:"job_spec_docker,omitempty" yaml:"job_spec_docker,omitempty"`
	Language JobSpecLanguage `json:"job_spec_language,omitempty" yaml:"job_spec_language,omitempty"`

	// the compute (cpy, ram) resources this job requires
	Resources ResourceUsageConfig `json:"resources" yaml:"resources"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	Inputs []StorageSpec `json:"inputs" yaml:"inputs"`

	// Input volumes that will not be sharded
	// for example to upload code into a base image
	// every shard will get the full range of context volumes
	Contexts []StorageSpec `json:"contexts,omitempty" yaml:"contexts,omitempty"`

	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []StorageSpec `json:"outputs" yaml:"outputs"`

	// Annotations on the job - could be user or machine assigned
	Annotations []string `json:"annotations" yaml:"annotations"`

	// the sharding config for this job
	// describes how the job might be split up into parallel shards
	Sharding JobShardingConfig `json:"sharding,omitempty" yaml:"sharding,omitempty"`

	// Do not track specified by the client
	DoNotTrack bool `json:"donottrack,omitempty" yaml:"donottrack,omitempty"`
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
	WorkingDir string `json:"workdir,omitempty" yaml:"workdir,omitempty"`
}

// for language style executors (can target docker or wasm)
type JobSpecLanguage struct {
	Language        string `json:"language" yaml:"language"`                 // e.g. python
	LanguageVersion string `json:"language_version" yaml:"language_version"` // e.g. 3.8
	// must this job be run in a deterministic context?
	Deterministic bool `json:"deterministic" yaml:"deterministic"`
	// context is a tar file stored in ipfs, containing e.g. source code and requirements
	Context StorageSpec `json:"context" yaml:"context"`
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
	JobDeal              JobDeal            `json:"job_deal"`
	Status               string             `json:"status"`
	RunOutput            *RunCommandResult  `json:"run_output"`
	VerificationProposal []byte             `json:"verification_proposal"`
	VerificationResult   VerificationResult `json:"verification_result"`
	PublishedResult      StorageSpec        `json:"published_results"`

	EventTime       time.Time `json:"event_time"`
	SenderPublicKey []byte    `json:"public_key"`
}

// we need to use a struct for the result because:
// a) otherwise we don't know if VerificationResult==false
// means "I've not verified yet" or "verification failed"
// b) we might want to add further fields to the result later
type VerificationResult struct {
	Complete bool `json:"complete"`
	Result   bool `json:"result"`
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

// JobStateType is the state of a job on a particular node. Note that the job
// will typically have different states on different nodes.
//
//go:generate stringer -type=JobStateType --trimprefix=JobState
type JobStateType int

// these are the states a job can be in against a single node
const (
	jobStateUnknown JobStateType = iota // must be first

	// a compute node has selected a job and has bid on it
	// we are currently waiting to hear back from the requester
	// node whether our bid was accepted or not
	JobStateBidding

	// a requester node has either rejected the bid or the compute node has canceled the bid
	// either way - this node will not progress with this job any more
	JobStateCancelled

	// the bid has been accepted but we have not yet started the job
	JobStateWaiting

	// the job is in the process of running
	JobStateRunning

	// the job had an error - this is an end state
	JobStateError

	// the compute node has finished execution and has communicated the ResultsProposal
	JobStateVerifying

	// our results have been processed and published
	JobStateCompleted

	jobStateDone // must be last
)

// IsTerminal returns true if the given job type signals the end of the
// lifecycle of that job on a particular node. After this, the job can be
// safely ignored by the node.
func (state JobStateType) IsTerminal() bool {
	return state == JobStateCompleted || state == JobStateError || state == JobStateCancelled
}

// IsComplete returns true if the given job has succeeded at the bid stage
// and has finished running the job - this is used to calculate if a job
// has completed across all nodes because a cancelation does not count
// towards actually "running" the job whereas an error does (even though it failed
// it still "ran")
func (state JobStateType) IsComplete() bool {
	return state == JobStateCompleted || state == JobStateError
}

func (state JobStateType) IsError() bool {
	return state == JobStateError
}

// tells you if this event is a valid one
func IsValidJobState(state JobStateType) bool {
	return state > jobStateUnknown && state < jobStateDone
}

func ParseJobStateType(str string) (JobStateType, error) {
	for typ := jobStateUnknown + 1; typ < jobStateDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return jobStateUnknown, fmt.Errorf(
		"executor: unknown job typ type '%s'", str)
}

func JobStateTypes() []JobStateType {
	var res []JobStateType
	for typ := jobStateUnknown + 1; typ < jobStateDone; typ++ {
		res = append(res, typ)
	}

	return res
}

// given an event name - return a job state
func GetStateFromEvent(eventType JobEventType) JobStateType {
	switch eventType {
	// we have bid and are waiting to hear if that has been accepted
	case JobEventBid:
		return JobStateBidding

	// our bid has been accepted but we've not yet started the job
	case JobEventBidAccepted:
		return JobStateWaiting

	// out bid got rejected so we are canceled
	case JobEventBidRejected:
		return JobStateCancelled

	// we canceled our bid so we are canceled
	case JobEventBidCancelled:
		return JobStateCancelled

	// we are running
	case JobEventRunning:
		return JobStateRunning

	// yikes
	case JobEventError:
		return JobStateError

	// we are complete
	case JobEventResultsProposed:
		return JobStateVerifying

	// both of these are "finalized"
	case JobEventResultsAccepted:
		return JobStateVerifying

	case JobEventResultsRejected:
		return JobStateVerifying

	case JobEventResultsPublished:
		return JobStateCompleted

	default:
		return jobStateUnknown
	}
}
