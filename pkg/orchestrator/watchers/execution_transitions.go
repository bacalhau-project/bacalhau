package watchers

import "github.com/bacalhau-project/bacalhau/pkg/models"

// executionTransitions helps determine valid state transitions for execution protocols
type executionTransitions struct {
	upsert models.ExecutionUpsert
}

func newExecutionTransitions(upsert models.ExecutionUpsert) *executionTransitions {
	return &executionTransitions{upsert: upsert}
}

// shouldAskForPendingBid returns true if we should request bid with approval:
// - New execution in pending state
// - Compute state is new
func (t *executionTransitions) shouldAskForPendingBid() bool {
	return t.upsert.Previous == nil &&
		t.upsert.Current.DesiredState.StateType == models.ExecutionDesiredStatePending
}

// shouldAskForDirectBid returns true if we should request immediate bid:
// - New execution in running state
// - Compute state is new
func (t *executionTransitions) shouldAskForDirectBid() bool {
	return t.upsert.Previous == nil &&
		t.upsert.Current.DesiredState.StateType == models.ExecutionDesiredStateRunning
}

// shouldAcceptBid returns true if bid should be accepted:
// - Already has a bid (in AskForBidAccepted state)
// - Moving to Running state from Pending
func (t *executionTransitions) shouldAcceptBid() bool {
	return t.upsert.Previous != nil &&
		t.upsert.Previous.DesiredState.StateType == models.ExecutionDesiredStatePending &&
		t.upsert.Current.DesiredState.StateType == models.ExecutionDesiredStateRunning
}

// shouldCancel returns true if we need to send a cancellation request when:
// 1. An execution exists (Previous is not nil)
// 2. The execution is transitioning to Stopped state from any non-Stopped state
// 3. The previous compute state is not terminal
//
// Note: We only check the previous compute state because the current state
// is always marked as terminal by the scheduler during cancellation. Checking
// the current state not terminal would cause us to incorrectly skip sending
// necessary cancellation requests.
func (t *executionTransitions) shouldCancel() bool {
	return t.upsert.Previous != nil &&
		t.upsert.Previous.DesiredState.StateType != models.ExecutionDesiredStateStopped &&
		t.upsert.Current.DesiredState.StateType == models.ExecutionDesiredStateStopped &&
		!t.upsert.Previous.IsTerminalComputeState()
}

// shouldRejectBid returns true if we need to send a bid rejection:
// - Moving from pending to stopped
// - Current state is one that can be rejected (new or bid accepted)
func (t *executionTransitions) shouldRejectBid() bool {
	return t.upsert.Previous != nil &&
		t.upsert.Previous.DesiredState.StateType == models.ExecutionDesiredStatePending &&
		t.upsert.Current.DesiredState.StateType == models.ExecutionDesiredStateStopped &&
		(t.upsert.Current.ComputeState.StateType == models.ExecutionStateAskForBid ||
			t.upsert.Current.ComputeState.StateType == models.ExecutionStateAskForBidAccepted)
}
