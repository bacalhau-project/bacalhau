package model

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
}

// ID returns the ID of the job.
func (j Job) ID() string {
	return j.Metadata.ID
}

// String returns the id of the job.
func (j Job) String() string {
	return j.Metadata.ID
}

type Metadata struct {
	// The unique global ID of this job in the bacalhau network.
	ID string `json:"ID,omitempty" example:"92d5d4ee-3765-4f78-8353-623f5f26df08"`

	// Time the job was submitted to the bacalhau network.
	CreatedAt time.Time `json:"CreatedAt,omitempty" example:"2022-11-17T13:29:01.871140291Z"`

	// The ID of the client that created this job.
	ClientID string `json:"ClientID,omitempty" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`

	Requester JobRequester `json:"Requester,omitempty"`
}
type JobRequester struct {
	// The ID of the requester node that owns this job.
	RequesterNodeID string `json:"RequesterNodeID,omitempty" example:"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF"`

	// The public key of the Requester node that created this job
	// This can be used to encrypt messages back to the creator
	RequesterPublicKey PublicKey `json:"RequesterPublicKey,omitempty"`
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
			Engine:   EngineDocker,
			Verifier: VerifierNoop,
			PublisherSpec: PublisherSpec{
				Type: PublisherEstuary,
			},
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
	// Job info
	Job Job `json:"Job"`
	// The current state of the job
	State JobState `json:"State"`
	// History of changes to the job state. Not always populated in the job description
	History []JobHistory `json:"History,omitempty"`
}

// The deal the client has made with the bacalhau network.
// This is updateable by the client who submitted the job
type Deal struct {
	// Whether the job should be run on any matching node (false) or all
	// matching nodes (true). If true, other fields in this struct are ignored.
	TargetAll bool `json:"TargetAll,omitempty"`
	// The maximum number of concurrent compute node bids that will be
	// accepted by the requester node on behalf of the client.
	Concurrency int `json:"Concurrency,omitempty"`
	// The number of nodes that must agree on a verification result
	// this is used by the different verifiers - for example the
	// deterministic verifier requires the winning group size
	// to be at least this size
	Confidence int `json:"Confidence,omitempty"`
	// The minimum number of bids that must be received before the Requester
	// node will randomly accept concurrency-many of them (when
	// TargetAll=false). This allows the Requester node to get some level of
	// guarantee that the execution of the jobs will be spread evenly across the
	// network (assuming that this value is some large proportion of the size of
	// the network).
	MinBids int `json:"MinBids,omitempty"`
}

// GetConcurrency returns the concurrency value from the deal
func (d Deal) GetConcurrency() int {
	if d.Concurrency == 0 {
		return 1
	}
	return d.Concurrency
}

// GetConfidence returns the confidence value from the deal
func (d Deal) GetConfidence() int {
	if d.Confidence == 0 {
		return d.GetConcurrency()
	}
	return d.Confidence
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

type PublisherSpec struct {
	Type   Publisher              `json:"Type,omitempty"`
	Params map[string]interface{} `json:"Params,omitempty"`
}

// Spec is a complete specification of a job that can be run on some
// execution provider.
type Spec struct {
	// e.g. docker or language
	Engine Engine `json:"Engine,omitempty"`

	Verifier Verifier `json:"Verifier,omitempty"`

	// there can be multiple publishers for the job
	// deprecated: use PublisherSpec instead
	Publisher     Publisher     `json:"Publisher,omitempty"`
	PublisherSpec PublisherSpec `json:"PublisherSpec,omitempty"`

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

	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []StorageSpec `json:"outputs,omitempty"`

	// Annotations on the job - could be user or machine assigned
	Annotations []string `json:"Annotations,omitempty"`

	// NodeSelectors is a selector which must be true for the compute node to run this job.
	NodeSelectors []LabelSelectorRequirement `json:"NodeSelectors,omitempty"`

	// Do not track specified by the client
	DoNotTrack bool `json:"DoNotTrack,omitempty"`

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

// we emit these to other nodes so they update their
// state locally and can emit events locally
type JobEvent struct {
	// APIVersion of the Job
	APIVersion string `json:"APIVersion,omitempty" example:"V1beta1"`

	JobID string `json:"JobID,omitempty" example:"9304c616-291f-41ad-b862-54e133c0149e"`
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

func (j JobCreatePayload) GetClientID() string {
	return j.ClientID
}

type JobCancelPayload struct {
	// the id of the client that is submitting the job
	ClientID string `json:"ClientID,omitempty" validate:"required"`

	// the job id of the job to be canceled
	JobID string `json:"JobID,omitempty" validate:"required"`

	// The reason that the job is being canceled
	Reason string `json:"Reason,omitempty"`
}

func (j JobCancelPayload) GetClientID() string {
	return j.ClientID
}

type LogsPayload struct {
	// the id of the client that is requesting the logs
	ClientID string `json:"ClientID,omitempty" validate:"required"`

	// the job id of the job to be shown
	JobID string `json:"JobID,omitempty" validate:"required"`

	// the execution to be shown
	ExecutionID string `json:"ExecutionID,omitempty" validate:"required"`

	// whether the logs history is required
	WithHistory bool `json:"WithHistory,omitempty"`

	// whether the logs should be followed after the current logs are shown
	Follow bool `json:"Follow,omitempty"`
}

func (j LogsPayload) GetClientID() string {
	return j.ClientID
}
