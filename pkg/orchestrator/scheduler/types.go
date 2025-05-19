package scheduler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// execSet is a set of executions with a series of helper functions defined
// that help reconcile state.
type execSet map[string]*models.Execution

//nolint:unused
func execSetFromSlice(executions []*models.Execution) execSet {
	set := execSet{}
	for _, exec := range executions {
		set[exec.ID] = exec
	}
	return set
}

func execSetFromSliceOfValues(executions []models.Execution) execSet {
	set := execSet{}
	for i, exec := range executions {
		set[exec.ID] = &executions[i]
	}
	return set
}

// String returns a string representation of the execution set.
func (set execSet) String() string {
	if len(set) == 0 {
		return "[]"
	}

	start := fmt.Sprintf("len(%d) [", len(set))
	var s []string
	for k, v := range set {
		s = append(s, fmt.Sprintf("%q: %v", k, v.ID))
	}
	return start + strings.Join(s, ", ") + "]"
}

// keys returns the keys of the set as a slice
//
//nolint:unused
func (set execSet) keys() []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	return keys
}

// ordered returns the executions in the set ordered by update time
func (set execSet) ordered() []*models.Execution {
	execs := make([]*models.Execution, 0, len(set))
	for _, alloc := range set {
		execs = append(execs, alloc)
	}
	sort.Slice(execs, func(i, j int) bool {
		return execs[i].ModifyTime < execs[j].ModifyTime
	})
	return execs
}

// filterNonTerminal filters out terminal execs
func (set execSet) filterNonTerminal() execSet {
	return set.filterBy(func(execution *models.Execution) bool {
		return !execution.IsTerminalState()
	})
}

// filterByState filters out execs that are not in the given state
func (set execSet) filterByState(state models.ExecutionStateType) execSet {
	return set.filterBy(func(execution *models.Execution) bool {
		return execution.ComputeState.StateType == state
	})
}

// filterByState filters out execs that were not initiate by the given evaluation ID
func (set execSet) filterByEvaluationID(evaluationID string) execSet {
	return set.filterBy(func(execution *models.Execution) bool {
		return execution.EvalID == evaluationID
	})
}

// runtimeID filters out execs with a given runtime ID
func (set execSet) filterByRuntimeID(runtimeID string) execSet {
	return set.filterBy(func(execution *models.Execution) bool {
		return execution.RuntimeID == runtimeID
	})
}

func (set execSet) excludeThisRuntimeID(runtimeID string) execSet {
	return set.filterBy(func(execution *models.Execution) bool {
		return execution.RuntimeID != runtimeID
	})
}

// filterByDesiredState filters out execs that are not in the given desired state
func (set execSet) filterByDesiredState(state models.ExecutionDesiredStateType) execSet {
	return set.filterBy(func(execution *models.Execution) bool {
		return execution.DesiredState.StateType == state
	})
}

// filterBy compute state filters out execs that don't match the given predicate
func (set execSet) filterBy(predicate func(execution *models.Execution) bool) execSet {
	filtered := execSet{}
	for _, exec := range set {
		if predicate(exec) {
			filtered[exec.ID] = exec
		}
	}
	return filtered
}

// filterFailed filters out non-failed executions.
func (set execSet) filterFailed() execSet {
	return set.filterByState(models.ExecutionStateFailed)
}

// filterCompleted filters out non-completed executions
func (set execSet) filterCompleted() execSet {
	return set.filterByState(models.ExecutionStateCompleted)
}

// groupByNodeHealth partitions executions based on their node's health status.
func (set execSet) groupByNodeHealth(nodeInfos map[string]*models.NodeInfo) (healthy execSet, lost execSet) {
	healthy = make(execSet)
	lost = make(execSet)
	for _, exec := range set {
		if _, ok := nodeInfos[exec.NodeID]; !ok {
			lost[exec.ID] = exec
			log.Debug().Msgf("Execution %s is running on node %s which is not healthy", exec.ID, exec.NodeID)
		} else {
			healthy[exec.ID] = exec
		}
	}
	return healthy, lost
}

// groupByExecutionTimeout partitions executions based on their timeout status.
func (set execSet) groupByExecutionTimeout(expirationTime time.Time) (remaining, timedOut execSet) {
	remaining = make(execSet)
	timedOut = make(execSet)
	for _, exec := range set {
		if exec.IsExpired(expirationTime) {
			timedOut[exec.ID] = exec
		} else {
			remaining[exec.ID] = exec
		}
	}
	return remaining, timedOut
}

// groupByPartition groups executions by their partition index, allowing operations
// to be performed independently on each partition's set of executions. This is crucial
// for maintaining partition isolation and ensuring correct scheduling behavior.
func (set execSet) groupByPartition() map[int]execSet {
	grouped := make(map[int]execSet)
	for _, exec := range set {
		if _, ok := grouped[exec.PartitionIndex]; !ok {
			grouped[exec.PartitionIndex] = make(execSet)
		}
		grouped[exec.PartitionIndex][exec.ID] = exec
	}
	return grouped
}

// completedPartitions returns a map of partition indices that have successfully completed.
// A partition is considered complete when at least one of its executions has reached
// the Completed state. This is used primarily for batch jobs to track overall progress.
func (set execSet) completedPartitions() map[int]bool {
	completed := make(map[int]bool)
	for _, exec := range set {
		if exec.ComputeState.StateType == models.ExecutionStateCompleted {
			completed[exec.PartitionIndex] = true
		}
	}
	return completed
}

// usedPartitions returns map of partition indices that have active (non-discarded) executions.
// An execution is considered "used" if it is either running or completed successfully.
// Failed, cancelled, or rejected executions are discarded and their partitions become
// available for retry. This is crucial for the retry mechanism to work correctly.
func (set execSet) usedPartitions() map[int]bool {
	used := make(map[int]bool)
	for _, exec := range set {
		if !exec.IsDiscarded() {
			used[exec.PartitionIndex] = true
		}
	}
	return used
}

// remainingPartitions returns slice of partition indices that need executions.
// A partition needs an execution if it has no active or completed executions.
// This happens either when:
// 1. The partition has never been assigned an execution
// 2. All previous executions for this partition have failed/been discarded
// The returned indices are used to schedule new executions for retry or initial scheduling.
func (set execSet) remainingPartitions(totalPartitions int) []int {
	used := set.usedPartitions()
	available := make([]int, 0, totalPartitions)
	for i := 0; i < totalPartitions; i++ {
		if !used[i] {
			available = append(available, i)
		}
	}
	return available
}

// executionsByApprovalStatus represents the different sets of executions based on their approval status.
type executionsByApprovalStatus struct {
	toApprove execSet
	toReject  execSet
	toCancel  execSet
}

// getApprovalStatuses evaluates executions for a single partition and determines which
// should be approved, rejected, or cancelled. The rules are:
// - If any execution is completed, all other executions are rejected/cancelled
// - If any execution is running (BidAccepted), keep the oldest and cancel others
// - Otherwise approve the oldest AskForBidAccepted execution and reject others
// This ensures exactly one active execution per partition at any time.
func (set execSet) getApprovalStatuses() executionsByApprovalStatus {
	result := executionsByApprovalStatus{
		toApprove: make(execSet),
		toReject:  make(execSet),
		toCancel:  make(execSet),
	}

	// If partition has a completed execution, reject/cancel all non-terminal
	if len(set.filterCompleted()) > 0 {
		for _, exec := range set {
			if exec.ComputeState.StateType == models.ExecutionStateAskForBidAccepted {
				result.toReject[exec.ID] = exec
			} else if !exec.IsTerminalState() {
				result.toCancel[exec.ID] = exec
			}
		}
		return result
	}

	//TODO: we are approving the oldest executions first, we should probably
	// approve the ones with highest rank first
	nonTerminalExecs := set.filterNonTerminal()
	runningExecs := nonTerminalExecs.filterByDesiredState(models.ExecutionDesiredStateRunning)
	orderedExecs := nonTerminalExecs.ordered()

	// If we have running executions, keep oldest and cancel/reject others
	if len(runningExecs) > 0 {
		var foundFirst bool
		for _, exec := range orderedExecs {
			// if the execution is running, keep the first one and cancel the rest
			if exec.DesiredState.StateType == models.ExecutionDesiredStateRunning && !foundFirst {
				foundFirst = true
			} else if exec.ComputeState.StateType == models.ExecutionStateAskForBidAccepted {
				result.toReject[exec.ID] = exec
			} else {
				result.toCancel[exec.ID] = exec
			}
		}
		return result
	}

	// No running executions - approve oldest eligible and reject other eligible ones
	var approved bool
	for _, exec := range orderedExecs {
		switch exec.ComputeState.StateType {
		case models.ExecutionStateAskForBidAccepted:
			if !approved {
				result.toApprove[exec.ID] = exec
				approved = true
			} else {
				result.toReject[exec.ID] = exec
			}
		default:
			// Do nothing for other states - they're not ready for approval/rejection
		}
	}

	return result
}

// markStopped
func (set execSet) markStopped(plan *models.Plan, event models.Event, computeState models.ExecutionStateType) {
	for _, exec := range set {
		plan.AppendStoppedExecution(exec, event, computeState)
	}
}

// markFailed
func (set execSet) markFailed(plan *models.Plan, event models.Event) {
	set.markStopped(plan, event, models.ExecutionStateFailed)
}

// markCancelled
func (set execSet) markCancelled(plan *models.Plan, event models.Event) {
	set.markStopped(plan, event, models.ExecutionStateCancelled)
}

// markRejected
func (set execSet) markRejected(plan *models.Plan, event models.Event) {
	set.markStopped(plan, event, models.ExecutionStateBidRejected)
}

// markApproved
func (set execSet) markApproved(plan *models.Plan, event models.Event) {
	for _, exec := range set {
		plan.AppendApprovedExecution(exec, event)
	}
}

// union returns the union of two sets. If there are any duplicates, the `other` will be used.
func (set execSet) union(other execSet) execSet {
	union := execSet{}
	for _, exec := range set {
		union[exec.ID] = exec
	}
	for _, exec := range other {
		union[exec.ID] = exec
	}
	return union
}
