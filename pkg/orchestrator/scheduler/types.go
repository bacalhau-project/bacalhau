package scheduler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// execSet is a set of executions with a series of helper functions defined
// that help reconcile state.
type execSet map[string]*model.ExecutionState

func execSetFromSlice(executions []*model.ExecutionState) execSet {
	set := execSet{}
	for _, exec := range executions {
		set[exec.ComputeReference] = exec
	}
	return set
}

func execSetFromSliceOfValues(executions []model.ExecutionState) execSet {
	set := execSet{}
	for i, exec := range executions {
		set[exec.ComputeReference] = &executions[i]
	}
	return set
}

// String returns a string representation of the execution set.
func (set execSet) String() string {
	if len(set) == 0 {
		return "[]"
	}

	start := fmt.Sprintf("len(%d) [\n", len(set))
	var s []string
	for k, v := range set {
		s = append(s, fmt.Sprintf("%q: %v", k, v.ComputeReference))
	}
	return start + strings.Join(s, "\n") + "]"
}

// has returns true if the set contains the given execution id
func (set execSet) has(key string) bool {
	_, ok := set[key]
	return ok
}

// keys returns the keys of the set as a slice
func (set execSet) keys() []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	return keys
}

// ordered returns the executions in the set ordered by update time
func (set execSet) ordered() []*model.ExecutionState {
	execs := make([]*model.ExecutionState, 0, len(set))
	for _, alloc := range set {
		execs = append(execs, alloc)
	}
	sort.Slice(execs, func(i, j int) bool {
		return execs[i].UpdateTime.Before(execs[j].UpdateTime)
	})
	return execs
}

// filterNonTerminal filters out terminal execs
func (set execSet) filterNonTerminal() execSet {
	filtered := execSet{}
	for _, exec := range set {
		if !exec.State.IsTerminal() {
			filtered[exec.ComputeReference] = exec
		}
	}
	return filtered
}

// filterByState filters out execs that are not in the given state
func (set execSet) filterByState(state model.ExecutionStateType) execSet {
	filtered := execSet{}
	for _, exec := range set {
		if exec.State == state {
			filtered[exec.ComputeReference] = exec
		}
	}
	return filtered
}

// filterRunning filters out non-running executions.
func (set execSet) filterRunning() execSet {
	return set.filterByState(model.ExecutionStateBidAccepted)
}

// filterFailed filters out non-failed executions.
func (set execSet) filterFailed() execSet {
	return set.filterByState(model.ExecutionStateFailed)
}

// union returns the union of two sets. If there are any duplicates, the `other` will be used.
func (set execSet) union(other execSet) execSet {
	union := execSet{}
	for _, exec := range set {
		union[exec.ComputeReference] = exec
	}
	for _, exec := range other {
		union[exec.ComputeReference] = exec
	}
	return union
}

// latest returns the latest execution in the set by the time it was last updated.
func (set execSet) latest() *model.ExecutionState {
	var latest *model.ExecutionState
	for _, exec := range set {
		if latest == nil || exec.UpdateTime.After(latest.UpdateTime) {
			latest = exec
		}
	}
	return latest
}

// countByState counts the number of executions in each state.
func (set execSet) countByState() map[model.ExecutionStateType]int {
	counts := map[model.ExecutionStateType]int{}
	for _, exec := range set {
		counts[exec.State]++
	}
	return counts
}

// countCompleted counts the number of completed executions.
func (set execSet) countCompleted() int {
	return set.countByState()[model.ExecutionStateCompleted]
}
