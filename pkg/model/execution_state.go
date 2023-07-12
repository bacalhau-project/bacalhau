//go:generate stringer -type=ExecutionStateType --trimprefix=ExecutionState --output execution_state_string.go
package model

import (
	"fmt"
	"time"
)

// ExecutionStateType The state of an execution. An execution represents a single attempt to execute a job on a node.
// A compute node can have multiple executions for the same job due to retries, but there can only be a single active execution
// per node at any given time.
type ExecutionStateType int

const (
	ExecutionStateNew ExecutionStateType = iota
	// ExecutionStateAskForBid A node has been selected to execute a job, and is being asked to bid on the job.
	ExecutionStateAskForBid
	// ExecutionStateAskForBidAccepted compute node has rejected the ask for bid.
	ExecutionStateAskForBidAccepted
	// ExecutionStateAskForBidRejected compute node has rejected the ask for bid.
	ExecutionStateAskForBidRejected
	// ExecutionStateBidAccepted requester has accepted the bid, and the execution is expected to be running on the compute node.
	ExecutionStateBidAccepted // aka running
	// ExecutionStateBidRejected requester has rejected the bid.
	ExecutionStateBidRejected
	// ExecutionStateCompleted The execution has been completed, and the result has been published.
	ExecutionStateCompleted
	// ExecutionStateFailed The execution has failed.
	ExecutionStateFailed
	// ExecutionStateCancelled The execution has been canceled by the user
	ExecutionStateCancelled
)

func ExecutionStateTypes() []ExecutionStateType {
	var res []ExecutionStateType
	for typ := ExecutionStateNew; typ <= ExecutionStateCancelled; typ++ {
		res = append(res, typ)
	}
	return res
}

// IsDiscarded returns true if the execution has been discarded due to a failure, rejection or cancellation
func (s ExecutionStateType) IsDiscarded() bool {
	return s == ExecutionStateAskForBidRejected || s == ExecutionStateBidRejected ||
		s == ExecutionStateCancelled || s == ExecutionStateFailed
}

// IsActive returns true if the execution is running or has completed
func (s ExecutionStateType) IsActive() bool {
	return s == ExecutionStateBidAccepted || s == ExecutionStateCompleted
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
	for typ := ExecutionStateNew; typ <= ExecutionStateCancelled; typ++ {
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
	NodeID      string `json:"NodeID,omitempty"`
	ExecutionID string `json:"ExecutionID,omitempty"`
}

// String returns a string representation of the execution id
func (e ExecutionID) String() string {
	return fmt.Sprintf("%s:%s:%s", e.JobID, ShortID(e.NodeID), e.ExecutionID)
}

type ExecutionState struct {
	// JobID the job id
	JobID string `json:"JobID"`
	// which node is running this execution
	NodeID string `json:"NodeId"`
	// Compute node reference for this job execution
	ComputeReference string `json:"ComputeReference"`
	// State is the current state of the execution
	State ExecutionStateType `json:"State"`
	// an arbitrary status message
	Status string `json:"Status,omitempty"`
	// the published results for this execution
	PublishedResult StorageSpec `json:"PublishedResults,omitempty"`

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
	return ExecutionID{JobID: e.JobID, NodeID: e.NodeID, ExecutionID: e.ComputeReference}
}

// String returns a string representation of the execution
func (e ExecutionState) String() string {
	return e.ID().String()
}
