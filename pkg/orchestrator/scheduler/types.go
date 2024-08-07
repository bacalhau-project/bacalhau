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

// has returns true if the set contains the given execution id
func (set execSet) has(key string) bool {
	_, ok := set[key]
	return ok
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
	filtered := execSet{}
	for _, exec := range set {
		if !exec.IsTerminalComputeState() {
			filtered[exec.ID] = exec
		}
	}
	return filtered
}

// filterByState filters out execs that are not in the given state
func (set execSet) filterByState(state models.ExecutionStateType) execSet {
	filtered := execSet{}
	for _, exec := range set {
		if exec.ComputeState.StateType == state {
			filtered[exec.ID] = exec
		}
	}
	return filtered
}

// filterRunning filters out non-running executions.
func (set execSet) filterRunning() execSet {
	return set.filterByState(models.ExecutionStateBidAccepted)
}

// filterFailed filters out non-failed executions.
func (set execSet) filterFailed() execSet {
	return set.filterByState(models.ExecutionStateFailed)
}

// filterOverSubscriptions partitions executions based on if they are more than the desired count.
func (set execSet) filterByOverSubscriptions(desiredCount int) (remaining execSet, overSubscriptions execSet) {
	remaining = make(execSet)
	overSubscriptions = make(execSet)
	count := 0
	for _, exec := range set.ordered() {
		if count >= desiredCount {
			overSubscriptions[exec.ID] = exec
		} else {
			remaining[exec.ID] = exec
		}
		count++
	}
	return remaining, overSubscriptions
}

// filterByNodeHealth partitions executions based on their node's health status.
func (set execSet) filterByNodeHealth(nodeInfos map[string]*models.NodeInfo) (healthy execSet, lost execSet) {
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

// filterByExecutionTimeout partitions executions based on their timeout status.
func (set execSet) filterByExecutionTimeout(expirationTime time.Time) (remaining, timedOut execSet) {
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

// executionsByApprovalStatus represents the different sets of executions based on their approval status.
type executionsByApprovalStatus struct {
	running   execSet
	toApprove execSet
	toReject  execSet
	pending   execSet
}

// activeCount returns the number of active executions, excluding rejected ones.
func (e executionsByApprovalStatus) activeCount() int {
	return len(e.running) + len(e.toApprove) + len(e.pending)
}

// filterByApprovalStatus partitions executions based on their approval status.
func (set execSet) filterByApprovalStatus(desiredCount int) executionsByApprovalStatus {
	nonTermExecs := set.filterNonTerminal()
	running := nonTermExecs.filterRunning()
	toApprove := make(execSet)
	toReject := make(execSet)
	pending := make(execSet)

	//TODO: we are approving the oldest executions first, we should probably
	// approve the ones with highest rank first
	orderedExecs := nonTermExecs.ordered()

	// Approve/Reject nodes
	for _, exec := range orderedExecs {
		// nothing left to approve
		if (len(running) + len(toApprove)) >= desiredCount {
			break
		}
		if exec.ComputeState.StateType == models.ExecutionStateAskForBidAccepted {
			toApprove[exec.ID] = exec
		}
	}

	// reject the rest
	totalRunningCount := len(running) + len(toApprove)
	for _, exec := range orderedExecs {
		if running.has(exec.ID) || toApprove.has(exec.ID) {
			continue
		}
		if totalRunningCount >= desiredCount {
			toReject[exec.ID] = exec
		} else {
			pending[exec.ID] = exec
		}
	}
	return executionsByApprovalStatus{
		running:   running,
		toApprove: toApprove,
		toReject:  toReject,
		pending:   pending,
	}
}

// markStopped
func (set execSet) markStopped(plan *models.Plan, event models.Event) {
	for _, exec := range set {
		plan.AppendStoppedExecution(exec, event)
	}
}

// markStopped
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

// countByState counts the number of executions in each state.
func (set execSet) countByState() map[models.ExecutionStateType]int {
	counts := map[models.ExecutionStateType]int{}
	for _, exec := range set {
		counts[exec.ComputeState.StateType]++
	}
	return counts
}

// countCompleted counts the number of completed executions.
func (set execSet) countCompleted() int {
	return set.countByState()[models.ExecutionStateCompleted]
}
