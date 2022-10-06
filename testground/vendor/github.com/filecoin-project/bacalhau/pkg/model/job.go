package model

import (
	"fmt"
	"time"

	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"
)

// Job contains data about a job request in the bacalhau network.
type Job struct {
	APIVersion string `json:"APIVersion"`

	// The unique global ID of this job in the bacalhau network.
	ID string `json:"ID,omitempty"`

	// The ID of the requester node that owns this job.
	RequesterNodeID string `json:"RequesterNodeID,omitempty"`

	// The public key of the requestor node that created this job
	// This can be used to encrypt messages back to the creator
	RequesterPublicKey PublicKey `json:"RequesterPublicKey,omitempty"`

	// The ID of the client that created this job.
	ClientID string `json:"ClientID,omitempty"`

	// The specification of this job.
	Spec Spec `json:"Spec,omitempty"`

	// The deal the client has made, such as which job bids they have accepted.
	Deal Deal `json:"Deal,omitempty"`

	// how will this job be executed by nodes on the network
	ExecutionPlan JobExecutionPlan `json:"ExecutionPlan,omitempty"`

	// Time the job was submitted to the bacalhau network.
	CreatedAt time.Time `json:"CreatedAt,omitempty"`

	// The current state of the job
	State JobState `json:"JobState,omitempty"`

	// All events associated with the job
	Events []JobEvent `json:"JobEvents,omitempty"`

	// All local events associated with the job
	LocalEvents []JobLocalEvent `json:"LocalJobEvents,omitempty"`
}

func (job Job) String() string {
	return job.ID
}

// TODO: There's probably a better way we want to globally version APIs
func NewJob() *Job {
	return &Job{
		APIVersion: APIVersionLatest().String(),
	}
}

func NewJobWithSaneProductionDefaults() (*Job, error) {
	j := NewJob()
	err := mergo.Merge(j, &Job{
		APIVersion: V1alpha1.String(),
		Spec: Spec{
			Engine:    EngineDocker,
			Verifier:  VerifierNoop,
			Publisher: PublisherEstuary,
		},
		Deal: Deal{
			Concurrency: 1,
			Confidence:  0,
			MinBids:     0, // 0 means no minimum before bidding
		},
	})
	if err != nil {
		log.Err(err).Msg("failed to merge sane defaults into job")
		return nil, err
	}
	return j, nil
}

// JobWithInfo is the job request + the result of attempting to run it on the network
type JobWithInfo struct {
	Job            Job             `json:"Job,omitempty"`
	JobState       JobState        `json:"JobState,omitempty"`
	JobEvents      []JobEvent      `json:"JobEvents,omitempty"`
	JobLocalEvents []JobLocalEvent `json:"JobLocalEvents,omitempty"`
}

// JobShard contains data about a job shard in the bacalhau network.
type JobShard struct {
	Job *Job `json:"Job,omitempty"`

	Index int `json:"Index,omitempty"`
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
	TotalShards int `json:"ShardsTotal,omitempty"`
}

// describe how we chunk a job up into shards
type JobShardingConfig struct {
	// divide the inputs up into the smallest possible unit
	// for example /* would mean "all top level files or folders"
	// this being an empty string means "no sharding"
	GlobPattern string `json:"GlobPattern,omitempty"`
	// how many "items" are to be processed in each shard
	// we first apply the glob pattern which will result in a flat list of items
	// this number decides how to group that flat list into actual shards run by compute nodes
	BatchSize int `json:"BatchSize,omitempty"`
	// when using multiple input volumes
	// what path do we treat as the common mount path to apply the glob pattern to
	BasePath string `json:"GlobPatternBasePath,omitempty"`
}

// The state of a job across the whole network
// generally be in different states on different nodes - one node may be
// ignoring a job as its bid was rejected, while another node may be
// submitting results for the job to the requester node.
//
// Each node will produce an array of JobShardState one for each shard
// (jobs without a sharding config will still have sharded job
// states - just with a shard count of 1). Any code that is determining
// the current "state" of a job must look at both:
//
//   - the ShardCount of the JobExecutionPlan
//   - the collection of JobShardState to determine the current state
//
// Note: JobState itself is not mutable - the JobExecutionPlan and
// JobShardState are updatable and the JobState is queried by the rest
// of the system.
type JobState struct {
	Nodes map[string]JobNodeState `json:"Nodes,omitempty"`
}

type JobNodeState struct {
	Shards map[int]JobShardState `json:"Shards,omitempty"`
}

type JobShardState struct {
	// which node is running this shard
	NodeID string `json:"NodeId,omitempty"`
	// what shard is this we are running
	ShardIndex int `json:"ShardIndex,omitempty"`
	// what is the state of the shard on this node
	State JobStateType `json:"State,omitempty"`
	// an arbitrary status message
	Status string `json:"Status,omitempty"`
	// the proposed results for this shard
	// this will be resolved by the verifier somehow
	VerificationProposal []byte             `json:"VerificationProposal,omitempty"`
	VerificationResult   VerificationResult `json:"VerificationResult,omitempty"`
	PublishedResult      StorageSpec        `json:"PublishedResults,omitempty"`

	// RunOutput of the job
	RunOutput *RunCommandResult `json:"RunOutput,omitempty"`
}

// The deal the client has made with the bacalhau network.
// This is updateable by the client who submitted the job
type Deal struct {
	// The maximum number of concurrent compute node bids that will be
	// accepted by the requester node on behalf of the client.
	Concurrency int `json:"Concurrency,omitempty"`
	// The number of nodes that must agree on a verification result
	// this is used by the different verifiers - for example the
	// deterministic verifier requires the winning group size
	// to be at least this size
	Confidence int `json:"Confidence,omitempty"`
	// The minimum number of bids that must be received before the requestor
	// node will randomly accept concurrency-many of them. This allows the
	// requestor node to get some level of guarantee that the execution of the
	// jobs will be spread evenly across the network (assuming that this value
	// is some large proportion of the size of the network).
	MinBids int `json:"MinBids,omitempty"`
}

// Spec is a complete specification of a job that can be run on some
// execution provider.
type Spec struct {
	// TODO: #643 #642 Merge EngineType & Engine, VerifierType & VerifierName, Publisher & PublisherName - this seems like an issue
	// e.g. docker or language
	Engine Engine `json:"Engine,omitempty"`
	// allow the engine to be provided as a string for JSON job specs
	EngineName string `json:"EngineName,omitempty"`

	Verifier Verifier `json:"Verifier,omitempty"`
	// allow the verifier to be provided as a string for JSON job specs
	VerifierName string `json:"VerifierName,omitempty"`

	// there can be multiple publishers for the job
	Publisher     Publisher `json:"Publisher,omitempty"`
	PublisherName string    `json:"PublisherName,omitempty"`

	// executor specific data
	Docker   JobSpecDocker   `json:"Docker,omitempty"`
	Language JobSpecLanguage `json:"Language,omitempty"`

	// the compute (cpy, ram) resources this job requires
	Resources ResourceUsageConfig `json:"Resources,omitempty"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	// TODO: #667 Replace with "Inputs", "Outputs" (note the caps) for yaml/json when we update the n.js file
	Inputs []StorageSpec `json:"inputs,omitempty"`

	// Input volumes that will not be sharded
	// for example to upload code into a base image
	// every shard will get the full range of context volumes
	Contexts []StorageSpec `json:"Contexts,omitempty"`

	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []StorageSpec `json:"outputs,omitempty"`

	// Annotations on the job - could be user or machine assigned
	Annotations []string `json:"Annotations,omitempty"`

	// the sharding config for this job
	// describes how the job might be split up into parallel shards
	Sharding JobShardingConfig `json:"Sharding,omitempty"`

	// Do not track specified by the client
	DoNotTrack bool `json:"DoNotTrack,omitempty"`
}

// for VM style executors
type JobSpecDocker struct {
	// this should be pullable by docker
	Image string `json:"Image,omitempty"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"Entrypoint,omitempty"`
	// a map of env to run the container with
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`
	// working directory inside the container
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

// for language style executors (can target docker or wasm)
type JobSpecLanguage struct {
	Language        string `json:"Language,omitempty"`        // e.g. python
	LanguageVersion string `json:"LanguageVersion,omitempty"` // e.g. 3.8
	// must this job be run in a deterministic context?
	Deterministic bool `json:"DeterministicExecution,omitempty"`
	// context is a tar file stored in ipfs, containing e.g. source code and requirements
	Context StorageSpec `json:"JobContext,omitempty"`
	// optional program specified on commandline, like python -c "print(1+1)"
	Command string `json:"Command,omitempty"`
	// optional program path relative to the context dir. one of Command or ProgramPath must be specified
	ProgramPath string `json:"ProgramPath,omitempty"`
	// optional requirements.txt (or equivalent) path relative to the context dir
	RequirementsPath string `json:"RequirementsPath,omitempty"`
}

// gives us a way to keep local data against a job
// so our compute node and requester node control loops
// can keep state against a job without broadcasting it
// to the rest of the network
type JobLocalEvent struct {
	EventName    JobLocalEventType `json:"EventName,omitempty"`
	JobID        string            `json:"JobID,omitempty"`
	ShardIndex   int               `json:"ShardIndex,omitempty"`
	TargetNodeID string            `json:"TargetNodeID,omitempty"`
}

// we emit these to other nodes so they update their
// state locally and can emit events locally
type JobEvent struct {
	// APIVersion of the Job
	APIVersion string `json:"APIVersion,omitempty"`

	JobID string `json:"JobID,omitempty"`
	// what shard is this event for
	ShardIndex int `json:"ShardIndex,omitempty"`
	// optional clientID if this is an externally triggered event (like create job)
	ClientID string `json:"ClientID,omitempty"`
	// the node that emitted this event
	SourceNodeID string `json:"SourceNodeID,omitempty"`
	// the node that this event is for
	// e.g. "AcceptJobBid" was emitted by requestor but it targeting compute node
	TargetNodeID string       `json:"TargetNodeID,omitempty"`
	EventName    JobEventType `json:"EventName,omitempty"`
	// this is only defined in "create" events
	Spec Spec `json:"Spec,omitempty"`
	// this is only defined in "create" events
	JobExecutionPlan JobExecutionPlan `json:"JobExecutionPlan,omitempty"`
	// this is only defined in "update_deal" events
	Deal                 Deal               `json:"Deal,omitempty"`
	Status               string             `json:"Status,omitempty"`
	VerificationProposal []byte             `json:"VerificationProposal,omitempty"`
	VerificationResult   VerificationResult `json:"VerificationResult,omitempty"`
	PublishedResult      StorageSpec        `json:"PublishedResult,omitempty"`

	EventTime       time.Time `json:"EventTime,omitempty"`
	SenderPublicKey PublicKey `json:"SenderPublicKey,omitempty"`

	// RunOutput of the job
	RunOutput *RunCommandResult `json:"RunOutput,omitempty"`
}

// we need to use a struct for the result because:
// a) otherwise we don't know if VerificationResult==false
// means "I've not verified yet" or "verification failed"
// b) we might want to add further fields to the result later
type VerificationResult struct {
	Complete bool `json:"Complete,omitempty"`
	Result   bool `json:"Result,omitempty"`
}

type JobCreatePayload struct {
	// the id of the client that is submitting the job
	ClientID string `json:"ClientID,omitempty"`

	// The job specification:
	Job *Job `json:"Job,omitempty"`

	// Optional base64-encoded tar file that will be pinned to IPFS and
	// mounted as storage for the job. Not part of the spec so we don't
	// flood the transport layer with it (potentially very large).
	Context string `json:"Context,omitempty"`
}
