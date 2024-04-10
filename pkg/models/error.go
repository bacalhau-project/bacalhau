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
	Hint() string
}

type HasRetryable interface {
	Retryable() bool
}

type HasFailsExecution interface {
	FailsExecution() bool
}

type HasDetails interface {
	Details() map[string]string
}

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
	if hintable, ok := err.(HasHint); ok {
		event.Details[DetailsKeyHint] = hintable.Hint()
	}
	if maybeRetryable, ok := err.(HasRetryable); ok {
		event.Details[DetailsKeyRetryable] = fmt.Sprint(maybeRetryable.Retryable())
	}
	if maybeFailsExecution, ok := err.(HasFailsExecution); ok {
		event.Details[DetailsKeyFailsExecution] = fmt.Sprint(maybeFailsExecution.FailsExecution())
	}

	event.Details[DetailsKeyIsError] = fmt.Sprint(true)
	return event
}
