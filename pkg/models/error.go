package models

import (
	"fmt"
	"maps"
	"time"
)

const (
	DetailsKeyIsError        = "IsError"
	DetailsKeyHint           = "Hint"
	DetailsKeyRetryable      = "Retryable"
	DetailsKeyFailsExecution = "FailsExecution"
)

type HasHint interface {
	// A human-readable string that advises the user on how they might solve the
	// error.
	Hint() string
}

type HasRetryable interface {
	// Whether or not the error could be retried, assuming the same input and
	// node configuration; i.e. the error is transient and due to network
	// capacity or service outage.
	//
	// If a component raises an error with Retryable() as true, the system may
	// retry the last action after some length of time. If it is false, it
	// should not try the action again, and may choose an alternative action or
	// fail the job.
	Retryable() bool
}

type HasFailsExecution interface {
	// Whether or not this error means that the associated execution cannot
	// continue.
	//
	// If a component raises an error with FailsExecution() as true,
	// the hosting executor should report the execution as failed and stop any
	// further steps.
	FailsExecution() bool
}

type HasDetails interface {
	// An extra set of metadata provided by the error.
	Details() map[string]string
}

// EventFromError converts an error into an Event tagged with the passed event
// topic.
//
// This method allows errors to implement extra interfaces (above) to do
// "attribute-based error reporting". The design principle is that errors can
// report a set of extra flags that have well defined semantics which the system
// can then respond to with specific behavior. This allows introducing or
// refactoring error types without higher-level components needing to be
// modified – they simply continue to respond to the presence of attributes.
//
// This is instead of the sysetm having a centralized set of known error types
// and programming in specific behavior in response to them, which is brittle
// and requires updating all of the error responses when the types change.
func EventFromError(topic EventTopic, err error) Event {
	event := Event{
		Message:   err.Error(),
		Timestamp: time.Now(),
		Topic:     topic,
		Details:   make(map[string]string, 4), //nolint:gomnd // number of inserts below
	}

	if hasDetails, ok := err.(HasDetails); ok {
		maps.Copy(event.Details, hasDetails.Details())
	}
	if hasHint, ok := err.(HasHint); ok {
		event.Details[DetailsKeyHint] = hasHint.Hint()
	}
	if hasRetryable, ok := err.(HasRetryable); ok {
		event.Details[DetailsKeyRetryable] = fmt.Sprint(hasRetryable.Retryable())
	}
	if hasFailsExecution, ok := err.(HasFailsExecution); ok {
		event.Details[DetailsKeyFailsExecution] = fmt.Sprint(hasFailsExecution.FailsExecution())
	}

	event.Details[DetailsKeyIsError] = fmt.Sprint(true)
	return event
}
