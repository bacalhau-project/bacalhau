//go:build unit || !integration

package scheduler

import (
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestExecSet_FilterNonTerminal(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
	}

	set := execSetFromSlice(executions)
	filtered := set.filterNonTerminal()
	assert.ElementsMatch(t, filtered.keys(), []string{"exec1"})
}

func TestExecSet_FilterByState(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec4", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
	}

	set := execSetFromSlice(executions)

	filtered1 := set.filterByState(models.ExecutionStateBidAccepted)
	assert.Len(t, filtered1, 2)
	assert.ElementsMatch(t, filtered1.keys(), []string{"exec1", "exec3"})

	filtered2 := set.filterByState(models.ExecutionStateFailed)
	assert.Len(t, filtered2, 1)
	assert.ElementsMatch(t, filtered2.keys(), []string{"exec2"})

	filtered3 := set.filterByState(models.ExecutionStateCompleted)
	assert.Len(t, filtered3, 1)
	assert.ElementsMatch(t, filtered3.keys(), []string{"exec4"})
}

func TestExecSet_FilterRunning(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted)},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
	}

	set := execSetFromSlice(executions)
	filtered := set.filterRunning()

	assert.Len(t, filtered, 1)
	assert.ElementsMatch(t, filtered.keys(), []string{"exec1"})
}

func TestExecSet_FilterFailed(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
	}

	set := execSetFromSlice(executions)
	filtered := set.filterFailed()

	assert.Len(t, filtered, 2)
	assert.ElementsMatch(t, filtered.keys(), []string{"exec1", "exec3"})
}

func TestExecSet_Union(t *testing.T) {
	set1 := execSet{
		"exec1": {ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateAskForBid)},
		"exec2": {ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
	}

	set2 := execSet{
		"exec2": {ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		"exec3": {ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
	}

	union := set1.union(set2)

	assert.Len(t, union, 3)
	assert.ElementsMatch(t, union.keys(), []string{"exec1", "exec2", "exec3"})

	// verify exec2 of the second set is the one that is kept
	assert.Equal(t, models.ExecutionStateCompleted, union["exec2"].ComputeState.StateType)
}

func TestExecSet_Latest(t *testing.T) {
	now := time.Now()
	executions := []*models.Execution{
		{ID: "exec1", ModifyTime: now.UnixNano()},
		{ID: "exec2", ModifyTime: now.Add(+1 * time.Second).UnixNano()},
		{ID: "exec3", ModifyTime: now.Add(-1 * time.Second).UnixNano()},
	}

	set := execSetFromSlice(executions)
	latest := set.latest()

	assert.NotNil(t, latest)
	assert.Equal(t, "exec2", latest.ID)
}

func TestExecSet_CountByState(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec4", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
	}

	set := execSetFromSlice(executions)
	counts := set.countByState()

	assert.Equal(t, 2, counts[models.ExecutionStateBidAccepted])
	assert.Equal(t, 1, counts[models.ExecutionStateFailed])
	assert.Equal(t, 1, counts[models.ExecutionStateCompleted])
}

func TestExecSet_CountCompleted(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec4", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
	}

	set := execSetFromSlice(executions)
	count := set.countCompleted()

	assert.Equal(t, 2, count)
}

func TestExecSet_String(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
	}

	set := execSetFromSlice(executions)
	str := set.String()

	assert.Contains(t, str, "len(3)")
	assert.Contains(t, str, `"exec1":`)
	assert.Contains(t, str, `"exec2":`)
	assert.Contains(t, str, `"exec3":`)
}

func TestExecSet_has(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
	}

	set := execSetFromSlice(executions)

	assert.True(t, set.has("exec1"))
	assert.True(t, set.has("exec2"))
	assert.False(t, set.has("exec3"))
}

func TestExecSet_FilterByNodeHealth(t *testing.T) {
	nodeInfos := map[string]*models.NodeInfo{
		"node1": {},
		"node2": {},
	}

	executions := []*models.Execution{
		{ID: "exec1", NodeID: "node1"},
		{ID: "exec2", NodeID: "node2"},
		{ID: "exec3", NodeID: "node3"},
	}

	set := execSetFromSlice(executions)
	healthy, lost := set.filterByNodeHealth(nodeInfos)

	assert.Len(t, healthy, 2)
	assert.Len(t, lost, 1)
	assert.ElementsMatch(t, healthy.keys(), []string{"exec1", "exec2"})
	assert.ElementsMatch(t, lost.keys(), []string{"exec3"})
}

func TestExecSet_FilterByOverSubscription(t *testing.T) {
	desiredCount := 3
	now := time.Now()

	executions := []*models.Execution{
		{ID: "exec1", ModifyTime: now.UnixNano()},
		{ID: "exec2", ModifyTime: now.Add(time.Second).UnixNano()},
		{ID: "exec3", ModifyTime: now.Add(2 * time.Second).UnixNano()},
		{ID: "exec4", ModifyTime: now.Add(3 * time.Second).UnixNano()},
		{ID: "exec5", ModifyTime: now.Add(4 * time.Second).UnixNano()},
	}

	set := execSetFromSlice(executions)
	remaining, overSubscriptions := set.filterByOverSubscriptions(desiredCount)

	assert.ElementsMatch(t, remaining.keys(), []string{"exec1", "exec2", "exec3"})
	assert.ElementsMatch(t, overSubscriptions.keys(), []string{"exec4", "exec5"})
}

func TestExecSet_FilterByApprovalStatus(t *testing.T) {
	desiredCount := 3
	now := time.Now()

	executions := []*models.Execution{
		{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted), ModifyTime: now.UnixNano()},
		{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted), ModifyTime: now.Add(time.Second).UnixNano()},
		{ID: "exec3", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted), ModifyTime: now.Add(2 * time.Second).UnixNano()},
		{ID: "exec4", ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted), ModifyTime: now.Add(3 * time.Second).UnixNano()},
		{ID: "exec5", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted), ModifyTime: now.Add(4 * time.Second).UnixNano()},
	}

	set := execSetFromSlice(executions)
	approvalStatus := set.filterByApprovalStatus(desiredCount)

	assert.ElementsMatch(t, approvalStatus.running.keys(), []string{"exec3", "exec4"})
	assert.ElementsMatch(t, approvalStatus.toApprove.keys(), []string{"exec1"})
	assert.ElementsMatch(t, approvalStatus.toReject.keys(), []string{"exec2"})
	assert.ElementsMatch(t, approvalStatus.pending.keys(), []string{})
	assert.Equal(t, 3, approvalStatus.activeCount())
}
