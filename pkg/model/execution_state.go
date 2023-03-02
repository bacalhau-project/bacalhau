package model

import (
	"fmt"
	"time"
)

// ExecutionStateType The state of an execution. An execution represents a single attempt to execute a shard on a node.
// A compute node can have multiple executions for the same shard due to retries, but there can only be a single active execution
// per node at any given time.
type ExecutionStateType int

const (
	ExecutionStateNew ExecutionStateType = iota
	// ExecutionStateAskForBid A node has been selected to execute a shard, and is being asked to bid on the shard.
	ExecutionStateAskForBid
	// ExecutionStateAskForBidAccepted compute node has rejected the ask for bid.
	ExecutionStateAskForBidAccepted
	// ExecutionStateAskForBidRejected compute node has rejected the ask for bid.
	ExecutionStateAskForBidRejected
	// ExecutionStateBidAccepted requester has accepted the bid, and the execution is expected to be running on the compute node.
	ExecutionStateBidAccepted // aka running
	// ExecutionStateBidRejected requester has rejected the bid.
	ExecutionStateBidRejected
	// ExecutionStateResultProposed The execution is done, and is waiting for verification.
	ExecutionStateResultProposed
	// ExecutionStateResultAccepted The execution result has been accepted by the requester, and publishing of the result is in progress.
	ExecutionStateResultAccepted // aka publishing
	// ExecutionStateResultRejected The execution result has been rejected by the requester.
	ExecutionStateResultRejected
	// ExecutionStateCompleted The execution has been completed, and the result has been published.
	ExecutionStateCompleted
	// ExecutionStateFailed The execution has failed.
	ExecutionStateFailed
	// ExecutionStateCanceled The execution has been canceled by the user
	ExecutionStateCanceled
)

func ExecutionStateTypes() []ExecutionStateType {
	var res []ExecutionStateType
	for typ := ExecutionStateNew; typ <= ExecutionStateCanceled; typ++ {
		res = append(res, typ)
	}
	return res
}

// IsDiscarded returns true if the execution has been discarded due to a failure, rejection or cancellation
func (s ExecutionStateType) IsDiscarded() bool {
	return s == ExecutionStateAskForBidRejected || s == ExecutionStateBidRejected || s == ExecutionStateResultRejected ||
		s == ExecutionStateCanceled || s == ExecutionStateFailed
}

// IsActive returns true if the execution is running or has completed
func (s ExecutionStateType) IsActive() bool {
	return s == ExecutionStateBidAccepted || s == ExecutionStateResultProposed ||
		s == ExecutionStateResultAccepted || s == ExecutionStateCompleted
}

// IsTerminal returns true if the execution is in a terminal state where no further state changes are possible
func (s ExecutionStateType) IsTerminal() bool {
	return s.IsDiscarded() || s == ExecutionStateCompleted
}

func (s ExecutionStateType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *ExecutionStateType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	for typ := ExecutionStateNew; typ <= ExecutionStateCanceled; typ++ {
		if equal(typ.String(), name) {
			*s = typ
			return
		}
	}
	return
}

// ExecutionID a globally unique identifier for an execution
type ExecutionID struct {
	JobID       string `json:"JobID,omitempty"`
	ShardIndex  int    `json:"ShardIndex,omitempty"`
	NodeID      string `json:"NodeID,omitempty"`
	ExecutionID string `json:"ExecutionID,omitempty"`
}

// String returns a string representation of the execution id
func (e ExecutionID) String() string {
	return fmt.Sprintf("%s:%d:%s:%s", e.JobID, e.ShardIndex, e.NodeID, e.ExecutionID)
}

// ShardID returns the shard ID for this execution id
func (e ExecutionID) ShardID() ShardID {
	return ShardID{JobID: e.JobID, Index: e.ShardIndex}
}

type ExecutionState struct {
	// JobID the job id
	JobID string `json:"JobID"`
	// what shard is this we are running
	ShardIndex int `json:"ShardIndex"`
	// which node is running this execution
	NodeID string `json:"NodeId"`
	// Compute node reference for this shard execution
	ComputeReference string `json:"ComputeReference"`
	// State is the current state of the execution
	State ExecutionStateType `json:"State"`
	// an arbitrary status message
	Status string `json:"Status,omitempty"`
	// the proposed results for this execution
	// this will be resolved by the verifier somehow
	VerificationProposal []byte             `json:"VerificationProposal,omitempty"`
	VerificationResult   VerificationResult `json:"VerificationResult,omitempty"`
	PublishedResult      StorageSpec        `json:"PublishedResults,omitempty"`

	// RunOutput of the job
	RunOutput *RunCommandResult `json:"RunOutput,omitempty"`
	// Version is the version of the job state. It is incremented every time the job state is updated.
	Version int `json:"Version"`
	// CreateTime is the time when the job was created.
	CreateTime time.Time `json:"CreateTime"`
	// UpdateTime is the time when the job state was last updated.
	UpdateTime time.Time `json:"UpdateTime"`
}

// ID returns the ID for this execution
func (e ExecutionState) ID() ExecutionID {
	return ExecutionID{JobID: e.JobID, ShardIndex: e.ShardIndex, NodeID: e.NodeID, ExecutionID: e.ComputeReference}
}

// ShardID returns the shard ID for this execution
func (e ExecutionState) ShardID() ShardID {
	return ShardID{JobID: e.JobID, Index: e.ShardIndex}
}

// String returns a string representation of the execution
func (e ExecutionState) String() string {
	return e.ID().String()
}

// HasAcceptedAskForBid returns true if the execution has been accepted by the node
// we rely on the value of the ExecutionID to determine if the askForBid has been accepted
func (e ExecutionState) HasAcceptedAskForBid() bool {
	return e.ComputeReference != ""
}
