package models

// State is a generic struct for representing the state of an object, with an
// optional human readable message.
type State[T any] struct {
	// StateType is the current state of the object.
	StateType T `json:"StateType"`

	// Message is a human readable message describing the state.
	Message string `json:"Message,omitempty"`

	// Details is a map of additional details about the state.
	Details map[string]string `json:"Details,omitempty"`
}

// WithMessage returns a new State with the specified message.
func (s State[T]) WithMessage(message string) State[T] {
	s.Message = message
	return s
}

// WithDetails returns a new State with the specified details.
func (s State[T]) WithDetails(details map[string]string) State[T] {
	if s.Details == nil {
		s.Details = make(map[string]string)
	}
	for k, v := range details {
		s.Details[k] = v
	}
	return s
}

// WithDetail returns a new State with the specified detail.
func (s State[T]) WithDetail(key, value string) State[T] {
	if s.Details == nil {
		s.Details = make(map[string]string)
	}
	s.Details[key] = value
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
