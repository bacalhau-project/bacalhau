package model

import (
	"fmt"
	"time"

	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"
)

// Job contains data about a job request in the bacalhau network.
type Job struct {
	APIVersion string `json:"APIVersion" yaml:"APIVersion"`

	// The unique global ID of this job in the bacalhau network.
	ID string `json:"ID,omitempty" yaml:"ID,omitempty"`

	// The ID of the requester node that owns this job.
	RequesterNodeID string `json:"RequesterNodeID,omitempty" yaml:"RequesterNodeID,omitempty"`

	// The public key of the requestor node that created this job
	// This can be used to encrypt messages back to the creator
	RequesterPublicKey PublicKey `json:"RequesterPublicKey,omitempty" yaml:"RequesterPublicKey,omitempty"`

	// The ID of the client that created this job.
	ClientID string `json:"ClientID,omitempty" yaml:"ClientID,omitempty"`

	// The specification of this job.
	Spec Spec `json:"Spec,omitempty" yaml:"Spec,omitempty"`

	// The deal the client has made, such as which job bids they have accepted.
	Deal Deal `json:"Deal,omitempty" yaml:"Deal,omitempty"`

	// how will this job be executed by nodes on the network
	ExecutionPlan JobExecutionPlan `json:"ExecutionPlan,omitempty" yaml:"ExecutionPlan,omitempty"`

	// Time the job was submitted to the bacalhau network.
	CreatedAt time.Time `json:"CreatedAt,omitempty" yaml:"CreatedAt,omitempty"`

	// The current state of the job
	State JobState `json:"JobState,omitempty" yaml:"JobState,omitempty"`

	// All events associated with the job
	Events []JobEvent `json:"JobEvents,omitempty" yaml:"JobEvents,omitempty"`

	// All local events associated with the job
	LocalEvents []JobLocalEvent `json:"LocalJobEvents,omitempty" yaml:"LocalJobEvents,omitempty"`
}

// TODO: There's probably a better way we want to globally version APIs
func NewJob() *Job {
	return &Job{
		APIVersion: V1alpha1.String(),
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
	Job            Job             `json:"Job,omitempty" yaml:"Job,omitempty"`
	JobState       JobState        `json:"JobState,omitempty" yaml:"JobState,omitempty"`
	JobEvents      []JobEvent      `json:"JobEvents,omitempty" yaml:"JobEvents,omitempty"`
	JobLocalEvents []JobLocalEvent `json:"JobLocalEvents,omitempty" yaml:"JobLocalEvents,omitempty"`
}

// JobShard contains data about a job shard in the bacalhau network.
type JobShard struct {
	Job *Job `json:"Job,omitempty" yaml:"Job,omitempty"`

	Index int `json:"Index,omitempty" yaml:"Index,omitempty"`
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
	TotalShards int `json:"ShardsTotal,omitempty" yaml:"ShardsTotal,omitempty"`
}

// describe how we chunk a job up into shards
type JobShardingConfig struct {
	// divide the inputs up into the smallest possible unit
	// for example /* would mean "all top level files or folders"
	// this being an empty string means "no sharding"
	GlobPattern string `json:"GlobPattern,omitempty" yaml:"GlobPattern,omitempty"`
	// how many "items" are to be processed in each shard
	// we first apply the glob pattern which will result in a flat list of items
	// this number decides how to group that flat list into actual shards run by compute nodes
	BatchSize int `json:"BatchSize,omitempty" yaml:"BatchSize,omitempty"`
	// when using multiple input volumes
	// what path do we treat as the common mount path to apply the glob pattern to
	BasePath string `json:"GlobPatternBasePath,omitempty" yaml:"GlobPatternBasePath,omitempty"`
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
	Nodes map[string]JobNodeState `json:"Nodes" yaml:"Nodes"`
}

type JobNodeState struct {
	Shards map[int]JobShardState `json:"Shards" yaml:"Shards"`
}

type JobShardState struct {
	// which node is running this shard
	NodeID string `json:"NodeId" yaml:"NodeId"`
	// what shard is this we are running
	ShardIndex int `json:"ShardIndex" yaml:"ShardIndex"`
	// what is the state of the shard on this node
	State JobStateType `json:"State" yaml:"State"`
	// an arbitrary status message
	Status string `json:"Status,omitempty" yaml:"Status,omitempty"`
	// the proposed results for this shard
	// this will be resolved by the verifier somehow
	VerificationProposal []byte             `json:"VerificationProposal,omitempty" yaml:"VerificationProposal,omitempty"`
	VerificationResult   VerificationResult `json:"VerificationResult,omitempty" yaml:"VerificationResult,omitempty"`
	PublishedResult      StorageSpec        `json:"PublishedResults,omitempty" yaml:"PublishedResults,omitempty"`

	// RunOutput of the job
	RunOutput *RunCommandResult `json:"RunOutput,omitempty" yaml:"RunOutput,omitempty"`
}

// The deal the client has made with the bacalhau network.
// This is updateable by the client who submitted the job
type Deal struct {
	// The maximum number of concurrent compute node bids that will be
	// accepted by the requester node on behalf of the client.
	Concurrency int `json:"Concurrency,omitempty" yaml:"Concurrency,omitempty"`
	// The number of nodes that must agree on a verification result
	// this is used by the different verifiers - for example the
	// deterministic verifier requires the winning group size
	// to be at least this size
	Confidence int `json:"Confidence,omitempty" yaml:"Confidence,omitempty"`
	// The minimum number of bids that must be received before the requestor
	// node will randomly accept concurrency-many of them. This allows the
	// requestor node to get some level of guarantee that the execution of the
	// jobs will be spread evenly across the network (assuming that this value
	// is some large proportion of the size of the network).
	MinBids int `json:"MinBids,omitempty" yaml:"MinBids,omitempty"`
}

// Spec is a complete specification of a job that can be run on some
// execution provider.
type Spec struct {
	// TODO: #643 #642 Merge EngineType & Engine, VerifierType & VerifierName, Publisher & PublisherName - this seems like an issue
	// e.g. docker or language
	Engine Engine `json:"Engine,omitempty" yaml:"Engine,omitempty"`
	// allow the engine to be provided as a string for yaml and JSON job specs
	EngineName string `json:"EngineName,omitempty" yaml:"EngineName,omitempty"`

	Verifier Verifier `json:"Verifier,omitempty" yaml:"Verifier,omitempty"`
	// allow the verifier to be provided as a string for yaml and JSON job specs
	VerifierName string `json:"VerifierName,omitempty" yaml:"VerifierName,omitempty"`

	// there can be multiple publishers for the job
	Publisher     Publisher `json:"Publisher,omitempty" yaml:"Publisher,omitempty"`
	PublisherName string    `json:"PublisherName,omitempty" yaml:"PublisherName,omitempty"`

	// executor specific data
	Docker   JobSpecDocker   `json:"Docker,omitempty" yaml:"Docker,omitempty"`
	Language JobSpecLanguage `json:"Language,omitempty" yaml:"Language,omitempty"`

	// the compute (cpy, ram) resources this job requires
	Resources ResourceUsageConfig `json:"Resources,omitempty" yaml:"Resources,omitempty"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	// TODO: #667 Replace with "Inputs", "Outputs" (note the caps) for yaml/json when we update the n.js file
	Inputs []StorageSpec `json:"inputs,omitempty" yaml:"inputs,omitempty"`

	// Input volumes that will not be sharded
	// for example to upload code into a base image
	// every shard will get the full range of context volumes
	Contexts []StorageSpec `json:"Contexts,omitempty" yaml:"Contexts,omitempty"`

	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []StorageSpec `json:"outputs,omitempty" yaml:"outputs,omitempty"`

	// Annotations on the job - could be user or machine assigned
	Annotations []string `json:"Annotations,omitempty" yaml:"Annotations,omitempty"`

	// the sharding config for this job
	// describes how the job might be split up into parallel shards
	Sharding JobShardingConfig `json:"Sharding,omitempty" yaml:"Sharding,omitempty"`

	// Do not track specified by the client
	DoNotTrack bool `json:"DoNotTrack,omitempty" yaml:"DoNotTrack,omitempty"`
}

// func (s *JobSpec) MarshalYAML() ([]byte, error) {
// 	type Alias JobSpec

// 	engineName, err := EnsureEngineType(s.Engine, s.EngineName)
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to marshal engine name")
// 		return nil, err
// 	}
// 	_ = engineName
// 	return json.Marshal(&struct {
// 		Engine     string `json:"Engine,omitempty" yaml:"Engine,omitempty"`
// 		EngineName string `json:"EngineName,omitempty" yaml:"EngineName,omitempty"`
// 		*Alias
// 	}{
// 		Engine:     "foo",
// 		EngineName: "",
// 	})
// }

// for VM style executors
type JobSpecDocker struct {
	// this should be pullable by docker
	Image string `json:"Image,omitempty" yaml:"Image,omitempty"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"Entrypoint,omitempty" yaml:"Entrypoint,omitempty"`
	// a map of env to run the container with
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty" yaml:"EnvironmentVariables,omitempty"`
	// working directory inside the container
	WorkingDirectory string `json:"WorkingDirectory,omitempty" yaml:"WorkingDirectory,omitempty"`
}

// for language style executors (can target docker or wasm)
type JobSpecLanguage struct {
	Language        string `json:"Language,omitempty" yaml:"Language,omitempty"`               // e.g. python
	LanguageVersion string `json:"LanguageVersion,omitempty" yaml:"LanguageVersion,omitempty"` // e.g. 3.8
	// must this job be run in a deterministic context?
	Deterministic bool `json:"DeterministicExecution,omitempty" yaml:"DeterministicExecution,omitempty"`
	// context is a tar file stored in ipfs, containing e.g. source code and requirements
	Context StorageSpec `json:"JobContext,omitempty" yaml:"JobContext,omitempty"`
	// optional program specified on commandline, like python -c "print(1+1)"
	Command string `json:"Command,omitempty" yaml:"Command,omitempty"`
	// optional program path relative to the context dir. one of Command or ProgramPath must be specified
	ProgramPath string `json:"ProgramPath,omitempty" yaml:"ProgramPath,omitempty"`
	// optional requirements.txt (or equivalent) path relative to the context dir
	RequirementsPath string `json:"RequirementsPath,omitempty" yaml:"RequirementsPath,omitempty"`
}

// gives us a way to keep local data against a job
// so our compute node and requester node control loops
// can keep state against a job without broadcasting it
// to the rest of the network
type JobLocalEvent struct {
	EventName    JobLocalEventType `json:"EventName" yaml:"EventName"`
	JobID        string            `json:"JobID" yaml:"JobID"`
	ShardIndex   int               `json:"ShardIndex" yaml:"ShardIndex"`
	TargetNodeID string            `json:"TargetNodeID,omitempty" yaml:"TargetNodeID,omitempty"`
}

// we emit these to other nodes so they update their
// state locally and can emit events locally
type JobEvent struct {
	JobID string `json:"JobID" yaml:"JobID"`
	// what shard is this event for
	ShardIndex int `json:"ShardIndex" yaml:"ShardIndex"`
	// optional clientID if this is an externally triggered event (like create job)
	ClientID string `json:"ClientID" yaml:"ClientID"`
	// the node that emitted this event
	SourceNodeID string `json:"SourceNodeID" yaml:"SourceNodeID"`
	// the node that this event is for
	// e.g. "AcceptJobBid" was emitted by requestor but it targeting compute node
	TargetNodeID string       `json:"TargetNodeID,omitempty" yaml:"TargetNodeID,omitempty"`
	EventName    JobEventType `json:"EventName" yaml:"EventName"`
	// this is only defined in "create" events
	Spec Spec `json:"Spec,omitempty" yaml:"Spec,omitempty"`
	// this is only defined in "create" events
	JobExecutionPlan JobExecutionPlan `json:"JobExecutionPlan" yaml:"JobExecutionPlan"`
	// this is only defined in "update_deal" events
	Deal                 Deal               `json:"Deal,omitempty" yaml:"Deal,omitempty"`
	Status               string             `json:"Status,omitempty" yaml:"Status,omitempty"`
	VerificationProposal []byte             `json:"VerificationProposal,omitempty" yaml:"VerificationProposal,omitempty"`
	VerificationResult   VerificationResult `json:"VerificationResult,omitempty" yaml:"VerificationResult,omitempty"`
	PublishedResult      StorageSpec        `json:"PublishedResult,omitempty" yaml:"PublishedResult,omitempty"`

	EventTime       time.Time `json:"EventTime" yaml:"EventTime"`
	SenderPublicKey PublicKey `json:"SenderPublicKey" yaml:"SenderPublicKey"`

	// RunOutput of the job
	RunOutput *RunCommandResult `json:"RunOutput,omitempty" yaml:"RunOutput,omitempty"`
}

// we need to use a struct for the result because:
// a) otherwise we don't know if VerificationResult==false
// means "I've not verified yet" or "verification failed"
// b) we might want to add further fields to the result later
type VerificationResult struct {
	Complete bool `json:"Complete" yaml:"Complete"`
	Result   bool `json:"Result" yaml:"Result"`
}

type JobCreatePayload struct {
	// the id of the client that is submitting the job
	ClientID string `json:"ClientID" yaml:"ClientID"`

	// The job specification:
	Job *Job `json:"Job" yaml:"Job"`

	// Optional base64-encoded tar file that will be pinned to IPFS and
	// mounted as storage for the job. Not part of the spec so we don't
	// flood the transport layer with it (potentially very large).
	Context string `json:"Context,omitempty" yaml:"Context,omitempty"`
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
