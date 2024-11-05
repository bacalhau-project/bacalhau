package models

type ExecutionUpsert struct {
	Current  *Execution
	Previous *Execution
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
