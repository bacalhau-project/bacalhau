package models

// State is a generic struct for representing the state of an object, with an
// optional human readable message.
type State[T any] struct {
	// State is the current state of the object.
	State T

	// Message is a human readable message describing the state.
	Message string
}
