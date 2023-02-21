package v1beta1

import (
	"time"

	"github.com/imdario/mergo"
	"k8s.io/apimachinery/pkg/selection"
)

// Job contains data about a job request in the bacalhau network.
type Job struct {
	APIVersion string `json:"APIVersion" example:"V1beta1"`

	Metadata Metadata `json:"Metadata,omitempty"`

	// The specification of this job.
	Spec Spec `json:"Spec,omitempty"`

	// The status of the job: where are the nodes at, what are the events
	Status JobStatus `json:"Status,omitempty"`
}

type Metadata struct {
	// The unique global ID of this job in the bacalhau network.
	ID string `json:"ID,omitempty" example:"92d5d4ee-3765-4f78-8353-623f5f26df08"`

	// Time the job was submitted to the bacalhau network.
	CreatedAt time.Time `json:"CreatedAt,omitempty" example:"2022-11-17T13:29:01.871140291Z"`

	// The ID of the client that created this job.
	ClientID string `json:"ClientID,omitempty" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
}
type JobRequester struct {
	// The ID of the requester node that owns this job.
	RequesterNodeID string `json:"RequesterNodeID,omitempty" example:"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF"`

	// The public key of the Requester node that created this job
	// This can be used to encrypt messages back to the creator
	RequesterPublicKey PublicKey `json:"RequesterPublicKey,omitempty"`
}
type JobStatus struct {
	// The current state of the job
	State JobState `json:"JobState,omitempty"`

	// All events associated with the job
	Events []JobEvent `json:"JobEvents,omitempty"`

	// All local events associated with the job
	LocalEvents []JobLocalEvent `json:"LocalJobEvents,omitempty"`

	Requester JobRequester `json:"Requester,omitempty"`
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
		APIVersion: APIVersionLatest().String(),
		Spec: Spec{
			Engine:    EngineDocker,
			Verifier:  VerifierNoop,
			Publisher: PublisherEstuary,
			Deal: Deal{
				Concurrency: 1,
				Confidence:  0,
				MinBids:     0, // 0 means no minimum before bidding
			},
		},
	})
	if err != nil {
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
	return GetShardID(shard.Job.Metadata.ID, shard.Index)
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
	// Compute node reference for this shard execution
	ExecutionID string `json:"ExecutionId,omitempty"`
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
	// The minimum number of bids that must be received before the Requester
	// node will randomly accept concurrency-many of them. This allows the
	// Requester node to get some level of guarantee that the execution of the
	// jobs will be spread evenly across the network (assuming that this value
	// is some large proportion of the size of the network).
	MinBids int `json:"MinBids,omitempty"`
}

// LabelSelectorRequirement A selector that contains values, a key, and an operator that relates the key and values.
// These are based on labels library from kubernetes package. While we use labels.Requirement to represent the label selector requirements
// in the command line arguments as the library supports multiple parsing formats, and we also use it when matching selectors to labels
// as that's what the library expects, labels.Requirements are not serializable, so we need to convert them to LabelSelectorRequirements.
type LabelSelectorRequirement struct {
	// key is the label key that the selector applies to.
	Key string `json:"Key"`
	// operator represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists and DoesNotExist.
	Operator selection.Operator `json:"Operator"`
	// values is an array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. This array is replaced during a strategic
	Values []string `json:"Values,omitempty"`
}

// Spec is a complete specification of a job that can be run on some
// execution provider.
type Spec struct {
	// e.g. docker or language
	Engine Engine `json:"Engine,omitempty"`

	Verifier Verifier `json:"Verifier,omitempty"`

	// there can be multiple publishers for the job
	Publisher Publisher `json:"Publisher,omitempty"`

	// executor specific data
	Docker   JobSpecDocker   `json:"Docker,omitempty"`
	Language JobSpecLanguage `json:"Language,omitempty"`
	Wasm     JobSpecWasm     `json:"Wasm,omitempty"`

	// the compute (cpu, ram) resources this job requires
	Resources ResourceUsageConfig `json:"Resources,omitempty"`

	// The type of networking access that the job needs
	Network NetworkConfig `json:"Network,omitempty"`

	// How long a job can run in seconds before it is killed.
	// This includes the time required to run, verify and publish results
	Timeout float64 `json:"Timeout,omitempty"`

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

	// NodeSelectors is a selector which must be true for the compute node to run this job.
	NodeSelectors []LabelSelectorRequirement `json:"NodeSelectors,omitempty"`
	// the sharding config for this job
	// describes how the job might be split up into parallel shards
	Sharding JobShardingConfig `json:"Sharding,omitempty"`

	// Do not track specified by the client
	DoNotTrack bool `json:"DoNotTrack,omitempty"`

	// how will this job be executed by nodes on the network
	ExecutionPlan JobExecutionPlan `json:"ExecutionPlan,omitempty"`

	// The deal the client has made, such as which job bids they have accepted.
	Deal Deal `json:"Deal,omitempty"`
}

// Return timeout duration
func (s *Spec) GetTimeout() time.Duration {
	return time.Duration(s.Timeout * float64(time.Second))
}

// Return pointers to all the storage specs in the spec.
func (s *Spec) AllStorageSpecs() []*StorageSpec {
	storages := []*StorageSpec{
		&s.Language.Context,
		&s.Wasm.EntryModule,
	}

	for _, collection := range [][]StorageSpec{
		s.Contexts,
		s.Inputs,
		s.Outputs,
	} {
		for index := range collection {
			storages = append(storages, &collection[index])
		}
	}

	return storages
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

// Describes a raw WASM job
type JobSpecWasm struct {
	// The module that contains the WASM code to start running.
	EntryModule StorageSpec `json:"EntryModule,omitempty"`

	// The name of the function in the EntryModule to call to run the job. For
	// WASI jobs, this will always be `_start`, but jobs can choose to call
	// other WASM functions instead. The EntryPoint must be a zero-parameter
	// zero-result function.
	EntryPoint string `json:"EntryPoint,omitempty"`

	// The arguments supplied to the program (i.e. as ARGV).
	Parameters []string `json:"Parameters,omitempty"`

	// The variables available in the environment of the running program.
	EnvironmentVariables map[string]string `json:"EnvironmentVariables,omitempty"`

	// TODO #880: Other WASM modules whose exports will be available as imports
	// to the EntryModule.
	ImportModules []StorageSpec `json:"ImportModules,omitempty"`
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
	APIVersion string `json:"APIVersion,omitempty" example:"V1beta1"`

	JobID string `json:"JobID,omitempty" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	// what shard is this event for
	ShardIndex int `json:"ShardIndex,omitempty"`
	// compute execution identifier
	ExecutionID string `json:"ExecutionID,omitempty" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	// optional clientID if this is an externally triggered event (like create job)
	ClientID string `json:"ClientID,omitempty" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	// the node that emitted this event
	SourceNodeID string `json:"SourceNodeID,omitempty" example:"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF"`
	// the node that this event is for
	// e.g. "AcceptJobBid" was emitted by Requester but it targeting compute node
	TargetNodeID string       `json:"TargetNodeID,omitempty" example:"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL"`
	EventName    JobEventType `json:"EventName,omitempty"`
	// this is only defined in "create" events
	Spec Spec `json:"Spec,omitempty"`
	// this is only defined in "create" events
	JobExecutionPlan JobExecutionPlan `json:"JobExecutionPlan,omitempty"`
	// this is only defined in "update_deal" events
	Deal                 Deal               `json:"Deal,omitempty"`
	Status               string             `json:"Status,omitempty" example:"Got results proposal of length: 0"`
	VerificationProposal []byte             `json:"VerificationProposal,omitempty"`
	VerificationResult   VerificationResult `json:"VerificationResult,omitempty"`
	PublishedResult      StorageSpec        `json:"PublishedResult,omitempty"`

	EventTime       time.Time `json:"EventTime,omitempty" example:"2022-11-17T13:32:55.756658941Z"`
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
	ClientID string `json:"ClientID,omitempty" validate:"required"`

	APIVersion string `json:"APIVersion,omitempty" example:"V1beta1" validate:"required"`

	// The specification of this job.
	Spec *Spec `json:"Spec,omitempty" validate:"required"`
}
