package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/imdario/mergo"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/selection"
)

// Job contains data about a job request in the bacalhau network.
type Job struct {
	APIVersion string `json:"APIVersion" example:"V1beta1"`

	// TODO this doesn't seem like it should be a part of the job as it cannot be known by a client ahead of time.
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

// TODO this must return a Noop EngineDeprecated.
func NewJobWithSaneProductionDefaults() (*Job, error) {
	j := NewJob()
	err := mergo.Merge(j, &Job{
		APIVersion: APIVersionLatest().String(),
		Spec: Spec{
			EngineDeprecated: EngineDocker,
			Verifier:         VerifierNoop,
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

type TargetingMode bool

const (
	TargetAny TargetingMode = false
	TargetAll TargetingMode = true
)

func (t TargetingMode) String() string {
	if bool(t) {
		return "all"
	} else {
		return "any"
	}
}

func ParseTargetingMode(s string) (TargetingMode, error) {
	switch s {
	case "any":
		return TargetAny, nil
	case "all":
		return TargetAll, nil
	default:
		return TargetAny, fmt.Errorf(`expecting "any" or "all", not %q`, s)
	}
}

// The deal the client has made with the bacalhau network.
// This is updateable by the client who submitted the job
type Deal struct {
	// Whether the job should be run on any matching node (false) or all
	// matching nodes (true). If true, other fields in this struct are ignored.
	TargetingMode TargetingMode `json:"TargetingMode,omitempty"`
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

func (d Deal) IsValid() error {
	var err error
	switch d.TargetingMode {
	case TargetAll:
		if d.Concurrency > 1 {
			// Although the requirement is stated as == 0, the default value is
			// 1, so we just ignore both 1 or 0 for convenience.
			err = multierr.Append(err, fmt.Errorf("concurrency ignored for target all mode, must be == 0"))
		}

		if d.Confidence != 0 {
			err = multierr.Append(err, fmt.Errorf("confidence ignored for target all mode, must be == 0"))
		}

		if d.MinBids != 0 {
			err = multierr.Append(err, fmt.Errorf("min bids ignored for target all mode, must be == 0"))
		}
	case TargetAny:
		if d.Concurrency <= 0 {
			err = multierr.Append(err, fmt.Errorf("concurrency must be >= 1"))
		}

		if d.Confidence < 0 {
			err = multierr.Append(err, fmt.Errorf("confidence must be >= 1"))
		}

		if d.MinBids < 0 {
			err = multierr.Append(err, fmt.Errorf("min bids must be >= 1"))
		}

		if d.Confidence > d.Concurrency {
			err = multierr.Append(err, fmt.Errorf("the deal confidence cannot be higher than the concurrency"))
		}
	}

	return err
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

func (r LabelSelectorRequirement) String() string {
	return fmt.Sprintf("%s %s %s", r.Key, r.Operator, strings.Join(r.Values, "|"))
}

type PublisherSpec struct {
	Type   Publisher              `json:"Type,omitempty"`
	Params map[string]interface{} `json:"Params,omitempty"`
}

type EngineSpec struct {
	Type   string                 `json:"Type,omitempty"`
	Params map[string]interface{} `json:"Params,omitempty"`
}

// Spec is a complete specification of a job that can be run on some
// execution provider.
type Spec struct {
	// deprecated: user EngineSpec instead.
	EngineDeprecated Engine `json:"Engine,omitempty"`
	EngineSpec       EngineSpec

	Verifier Verifier `json:"Verifier,omitempty"`

	// there can be multiple publishers for the job
	// deprecated: use PublisherSpec instead
	Publisher     Publisher     `json:"Publisher,omitempty"`
	PublisherSpec PublisherSpec `json:"PublisherSpec,omitempty"`

	// the compute (cpu, ram) resources this job requires
	Resources ResourceUsageConfig `json:"Resources,omitempty"`

	// The type of networking access that the job needs
	Network NetworkConfig `json:"Network,omitempty"`

	// How long a job can run in seconds before it is killed.
	// This includes the time required to run, verify and publish results
	Timeout float64 `json:"Timeout,omitempty"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	Inputs []StorageSpec `json:"Inputs,omitempty"`

	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []StorageSpec `json:"Outputs,omitempty"`

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
	storages := []*StorageSpec{}
	if s.EngineSpec.Type == EngineTypeWasm {
		wasmEngine, err := WasmEngineSpecFromEngineSpec(s.EngineSpec)
		if err != nil {
			panic(err)
		}
		storages = append(storages, &wasmEngine.EntryModule)
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
