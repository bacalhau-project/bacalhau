//go:generate stringer -type=ExecutionStateType --trimprefix=ExecutionState --output execution_state_string.go
//go:generate stringer -type=ExecutionDesiredStateType --trimprefix=ExecutionDesiredState --output execution_desired_state_string.go
package models

// ExecutionStateType The state of an execution. An execution represents a single attempt to execute a job on a node.
// A compute node can have multiple executions for the same job due to retries, but there can only be a single active execution
// per node at any given time.
type ExecutionStateType int

// TODO: change states to reflect non-bidding scheduling
const (
	ExecutionStateUndefined ExecutionStateType = iota
	// ExecutionStateNew The execution has been created, but not pushed to a compute node yet.
	ExecutionStateNew
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

type ExecutionDesiredStateType int

const (
	ExecutionDesiredStatePending ExecutionDesiredStateType = iota
	ExecutionDesiredStateRunning
	ExecutionDesiredStateStopped
)

// Execution is used to allocate the placement of a task group to a node.
type Execution struct {
	// ID of the execution (UUID)
	ID string

	// Namespace is the namespace the execution is created in
	Namespace string

	// ID of the evaluation that generated this execution
	EvalID string

	// Name is a logical name of the execution.
	Name string

	// NodeID is the node this is being placed on
	NodeID string

	// Job is the parent job of the task being allocated.
	// This is copied at execution time to avoid issues if the job
	// definition is updated.
	JobID string
	// TODO: evaluate using a copy of the job instead of a pointer
	Job *Job

	// AllocatedResources is the total resources allocated for the execution tasks.
	AllocatedResources *AllocatedResources

	// DesiredState of the execution on the compute node
	DesiredState State[ExecutionDesiredStateType]

	// ComputeState observed state of the execution on the compute node
	ComputeState State[ExecutionStateType]

	// the published results for this execution
	PublishedResult *SpecConfig

	// PreviousExecution is the execution that this execution is replacing
	PreviousExecution string

	// NextExecution is the execution that this execution is being replaced by
	NextExecution string

	// FollowupEvalID captures a follow up evaluation created to handle a failed execution
	// that can be rescheduled in the future
	FollowupEvalID string

	// Version is increment each time the execution is updated.
	Version uint64

	// CreateTime is the time the execution has finished scheduling and been
	// verified by the plan applier.
	CreateTime int64
	// ModifyTime is the time the execution was last updated.
	ModifyTime int64
}

func (a *Execution) JobNamespacedID() NamespacedID {
	return NewNamespacedID(a.JobID, a.Namespace)
}

// Normalize Allocation to ensure fields are initialized to the expectations
// of this version of Nomad. Should be called when restoring persisted
// Allocations or receiving Allocations from Nomad agents potentially on an
// older version of Nomad.
func (a *Execution) Normalize() {
	a.Job.Normalize()
}

// Copy provides a copy of the allocation and deep copies the job
func (a *Execution) Copy() *Execution {
	if a == nil {
		return nil
	}
	na := new(Execution)
	*na = *a

	na.Job = na.Job.Copy()
	na.AllocatedResources = na.AllocatedResources.Copy()
	na.PublishedResult = na.PublishedResult.Copy()
	return na
}
