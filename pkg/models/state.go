package models

// State is a generic struct for representing the state of an object, with an
// optional human readable message.
type State[T any] struct {
	// StateType is the current state of the object.
	StateType T `json:"StateType"`

	// Message is a human readable message describing the state.
	Message string `json:"Message,omitempty"`
}

// WithMessage returns a new State with the specified message.
func (s State[T]) WithMessage(message string) State[T] {
	s.Message = message
	return s
}

// NewJobState returns a new JobState with the specified state type
func NewJobState(stateType JobStateType) State[JobStateType] {
	return State[JobStateType]{
		StateType: stateType,
	}
}

// NewExecutionState returns a new ExecutionState with the specified state type
func NewExecutionState(stateType ExecutionStateType) State[ExecutionStateType] {
	return State[ExecutionStateType]{
		StateType: stateType,
	}
}

// NewExecutionDesiredState returns a new ExecutionDesiredStateType with the specified state type
func NewExecutionDesiredState(stateType ExecutionDesiredStateType) State[ExecutionDesiredStateType] {
	return State[ExecutionDesiredStateType]{
		StateType: stateType,
	}
}
