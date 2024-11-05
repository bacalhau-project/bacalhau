package models

// ExecutionUpsert represents a change in execution state, containing the current and previous
// execution states along with associated events. It is used for tracking and propagating
// execution state changes across nodes.
type ExecutionUpsert struct {
    // Current represents the new state of the execution
    Current  *Execution
    // Previous represents the old state of the execution, nil if this is a new execution
    Previous *Execution
    // Events contains the list of events associated with this state change
    Events   []*Event
}

// HasStateChange returns true if there are changes in either desired or compute state
func (u ExecutionUpsert) HasStateChange() bool {
	if u.Previous == nil {
		return true // new execution always counts as a state change
	}
	return u.Previous.DesiredState.StateType != u.Current.DesiredState.StateType ||
		u.Previous.ComputeState.StateType != u.Current.ComputeState.StateType
}
