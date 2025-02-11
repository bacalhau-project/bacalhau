//go:build unit || !integration

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	healthy, lost := set.groupByNodeHealth(nodeInfos)

	assert.Len(t, healthy, 2)
	assert.Len(t, lost, 1)
	assert.ElementsMatch(t, healthy.keys(), []string{"exec1", "exec2"})
	assert.ElementsMatch(t, lost.keys(), []string{"exec3"})
}

func TestExecSet_GetApprovalStatuses(t *testing.T) {
	t.Run("with completed execution", func(t *testing.T) {
		executions := []*models.Execution{
			{ID: "exec1", ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
			{ID: "exec2", ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted)},
			{ID: "exec3",
				ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
		}

		set := execSetFromSlice(executions)
		status := set.getApprovalStatuses()

		assert.Empty(t, status.toApprove)
		assert.ElementsMatch(t, status.toReject.keys(), []string{"exec2"})
		assert.ElementsMatch(t, status.toCancel.keys(), []string{"exec3"})
	})

	t.Run("with single running execution", func(t *testing.T) {
		now := time.Now()
		executions := []*models.Execution{
			{ID: "exec1",
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				ModifyTime:   now.UnixNano()},
		}

		set := execSetFromSlice(executions)
		status := set.getApprovalStatuses()

		assert.Empty(t, status.toApprove)
		assert.Empty(t, status.toReject)
		assert.Empty(t, status.toCancel)
	})

	t.Run("with multiple executions and one running", func(t *testing.T) {
		now := time.Now()
		executions := []*models.Execution{
			{ID: "exec1",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
				ModifyTime:   now.UnixNano()},
			{ID: "exec2",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
				ModifyTime:   now.Add(time.Second).UnixNano()},
			{ID: "exec3",
				ComputeState: models.NewExecutionState(models.ExecutionStateNew),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
				ModifyTime:   now.Add(2 * time.Second).UnixNano()},
		}

		set := execSetFromSlice(executions)
		status := set.getApprovalStatuses()

		assert.Empty(t, status.toApprove)
		assert.ElementsMatch(t, status.toReject.keys(), []string{"exec2"})
		assert.ElementsMatch(t, status.toCancel.keys(), []string{"exec3"})
	})

	t.Run("with multiple running executions preserves oldest", func(t *testing.T) {
		now := time.Now()
		executions := []*models.Execution{
			{ID: "exec1",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
				ModifyTime:   now.UnixNano()},
			{ID: "exec2",
				ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
				ModifyTime:   now.Add(time.Second).UnixNano()},
			{ID: "exec3",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
				ModifyTime:   now.Add(2 * time.Second).UnixNano()},
		}

		set := execSetFromSlice(executions)
		status := set.getApprovalStatuses()

		assert.Empty(t, status.toApprove)
		assert.ElementsMatch(t, status.toReject.keys(), []string{"exec3"})
		assert.ElementsMatch(t, status.toCancel.keys(), []string{"exec2"})
	})

	t.Run("with only pending executions", func(t *testing.T) {
		now := time.Now()
		executions := []*models.Execution{
			{ID: "exec1",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
				ModifyTime:   now.UnixNano()},
			{ID: "exec2",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
				ModifyTime:   now.Add(time.Second).UnixNano()},
		}

		set := execSetFromSlice(executions)
		status := set.getApprovalStatuses()

		assert.ElementsMatch(t, status.toApprove.keys(), []string{"exec1"})
		assert.ElementsMatch(t, status.toReject.keys(), []string{"exec2"})
		assert.Empty(t, status.toCancel)
	})

	t.Run("with mix of states and desired states", func(t *testing.T) {
		now := time.Now()
		executions := []*models.Execution{
			{ID: "exec1",
				ComputeState: models.NewExecutionState(models.ExecutionStateNew),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
				ModifyTime:   now.UnixNano()},
			{ID: "exec2",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
				ModifyTime:   now.Add(time.Second).UnixNano()},
			{ID: "exec3",
				ComputeState: models.NewExecutionState(models.ExecutionStateAskForBidAccepted),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
				ModifyTime:   now.Add(2 * time.Second).UnixNano()},
			{ID: "exec4",
				ComputeState: models.NewExecutionState(models.ExecutionStateFailed),
				DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped),
				ModifyTime:   now.Add(3 * time.Second).UnixNano()},
		}

		set := execSetFromSlice(executions)
		status := set.getApprovalStatuses()

		assert.Empty(t, status.toApprove)
		assert.ElementsMatch(t, status.toReject.keys(), []string{"exec3"})
		assert.ElementsMatch(t, status.toCancel.keys(), []string{"exec1"})
	})
}
func TestExecSet_FilterByExecutionTimeout(t *testing.T) {
	// Create a set of executions with varying execution times
	now := time.Now()

	executions := []*models.Execution{
		{ID: "exec1", ModifyTime: now.Add(-30 * time.Minute).UnixNano(), ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", ModifyTime: now.Add(-90 * time.Minute).UnixNano(), ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec3", ModifyTime: now.Add(-120 * time.Minute).UnixNano(), ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
	}
	execs := execSetFromSlice(executions)

	// Define an expiration timeout of 60 minutes in the past
	expirationTime := now.Add(-60 * time.Minute)

	// Filter executions by timeout
	remainingExecs, timedOutExecs := execs.groupByExecutionTimeout(expirationTime)

	// Check that the executions that have not exceeded the timeout remain in the set
	assert.Len(t, remainingExecs, 1)
	assert.Contains(t, remainingExecs, "exec1")

	// Check that the executions that have exceeded the timeout are correctly identified
	assert.Len(t, timedOutExecs, 2)
	assert.Contains(t, timedOutExecs, "exec2")
	assert.Contains(t, timedOutExecs, "exec3")
}

func TestExecSet_GroupByPartition(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", PartitionIndex: 0, ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", PartitionIndex: 1, ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec3", PartitionIndex: 0, ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec4", PartitionIndex: 2, ComputeState: models.NewExecutionState(models.ExecutionStateRunning)},
	}

	set := execSetFromSlice(executions)
	groups := set.groupByPartition()

	assert.Len(t, groups, 3)    // Should have 3 partitions
	assert.Len(t, groups[0], 2) // Partition 0 has 2 executions
	assert.Len(t, groups[1], 1) // Partition 1 has 1 execution
	assert.Len(t, groups[2], 1) // Partition 2 has 1 execution

	// Verify executions in partition 0
	assert.ElementsMatch(t, groups[0].keys(), []string{"exec1", "exec3"})
}

func TestExecSet_CompletedPartitions(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", PartitionIndex: 0, ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", PartitionIndex: 1, ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec3", PartitionIndex: 0, ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec4", PartitionIndex: 2, ComputeState: models.NewExecutionState(models.ExecutionStateRunning)},
	}

	set := execSetFromSlice(executions)
	completed := set.completedPartitions()

	assert.Len(t, completed, 2)   // Should have 2 completed partitions
	assert.True(t, completed[0])  // Partition 0 is completed
	assert.True(t, completed[1])  // Partition 1 is completed
	assert.False(t, completed[2]) // Partition 2 is not completed
}

func TestExecSet_UsedPartitions(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", PartitionIndex: 0, ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", PartitionIndex: 1, ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec3", PartitionIndex: 2, ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec4", PartitionIndex: 3, ComputeState: models.NewExecutionState(models.ExecutionStateCancelled)},
	}

	set := execSetFromSlice(executions)
	used := set.usedPartitions()

	assert.Len(t, used, 2)   // Should have 2 used partitions (running and completed)
	assert.True(t, used[0])  // Partition 0 is used (running)
	assert.False(t, used[1]) // Partition 1 is not used (failed)
	assert.True(t, used[2])  // Partition 2 is used (completed)
	assert.False(t, used[3]) // Partition 3 is not used (cancelled)
}

func TestExecSet_RemainingPartitions(t *testing.T) {
	executions := []*models.Execution{
		{ID: "exec1", PartitionIndex: 0, ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted)},
		{ID: "exec2", PartitionIndex: 1, ComputeState: models.NewExecutionState(models.ExecutionStateFailed)},
		{ID: "exec3", PartitionIndex: 2, ComputeState: models.NewExecutionState(models.ExecutionStateCompleted)},
		{ID: "exec4", PartitionIndex: 3, ComputeState: models.NewExecutionState(models.ExecutionStateCancelled)},
	}

	set := execSetFromSlice(executions)

	// Test with total partitions = 5
	remaining := set.remainingPartitions(5)
	assert.Len(t, remaining, 3) // Should have partitions 1, 3, and 4 remaining
	assert.ElementsMatch(t, remaining, []int{1, 3, 4})

	// Test with exact count
	remaining = set.remainingPartitions(4)
	assert.Len(t, remaining, 2) // Should have partitions 1 and 3 remaining
	assert.ElementsMatch(t, remaining, []int{1, 3})
}
