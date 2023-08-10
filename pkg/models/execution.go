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

// IsUndefined returns true if the execution state is undefined
func (s ExecutionStateType) IsUndefined() bool {
	return s == ExecutionStateUndefined
}

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

	// RunOutput is the output of the run command
	// TODO: evaluate removing this from execution spec in favour of calling `bacalhau logs`
	RunOutput *RunCommandResult

	// PreviousExecution is the execution that this execution is replacing
	PreviousExecution string

	// NextExecution is the execution that this execution is being replaced by
	NextExecution string

	// FollowupEvalID captures a follow up evaluation created to handle a failed execution
	// that can be rescheduled in the future
	FollowupEvalID string

	// Revision is increment each time the execution is updated.
	Revision uint64

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

// IsTerminalState returns true if the execution desired of observed state is terminal
func (a *Execution) IsTerminalState() bool {
	return a.IsTerminalDesiredState() || a.IsTerminalComputeState()
}

// IsTerminalDesiredState returns true if the execution desired state is terminal
func (a *Execution) IsTerminalDesiredState() bool {
	return a.DesiredState.StateType == ExecutionDesiredStateStopped
}

// IsTerminalComputeState returns true if the execution observed state is terminal
func (a *Execution) IsTerminalComputeState() bool {
	switch a.ComputeState.StateType {
	case ExecutionStateCompleted, ExecutionStateFailed, ExecutionStateCancelled, ExecutionStateAskForBidRejected, ExecutionStateBidRejected:
		return true
	default:
		return false
	}
}

// IsDiscarded returns true if the execution has failed, been cancelled or rejected.
func (a *Execution) IsDiscarded() bool {
	switch a.ComputeState.StateType {
	case ExecutionStateAskForBidRejected, ExecutionStateBidRejected, ExecutionStateCancelled, ExecutionStateFailed:
		return true
	default:
		return false
	}
}

type RunCommandResult struct {
	// stdout of the run. Yaml provided for `describe` output
	STDOUT string `json:"stdout"`

	// bool describing if stdout was truncated
	StdoutTruncated bool `json:"stdouttruncated"`

	// stderr of the run.
	STDERR string `json:"stderr"`

	// bool describing if stderr was truncated
	StderrTruncated bool `json:"stderrtruncated"`

	// exit code of the run.
	ExitCode int `json:"exitCode"`

	// Runner error
	ErrorMsg string `json:"runnerError"`
}

func NewRunCommandResult() *RunCommandResult {
	return &RunCommandResult{
		STDOUT:          "",    // stdout of the run.
		StdoutTruncated: false, // bool describing if stdout was truncated
		STDERR:          "",    // stderr of the run.
		StderrTruncated: false, // bool describing if stderr was truncated
		ExitCode:        -1,    // exit code of the run.
	}
}
