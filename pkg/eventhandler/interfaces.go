package eventhandler

//go:generate mockgen --source interfaces.go --destination mock_eventhandler/mock_handlers.go --package mock_eventhandler

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// A job event handler is a component that is notified of events related to jobs.
type JobEventHandler interface {
	HandleJobEvent(ctx context.Context, event model.JobEvent) error
}

// function that implements the JobEventHandler interface
type JobEventHandlerFunc func(ctx context.Context, event model.JobEvent) error

func (f JobEventHandlerFunc) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	return f(ctx, event)
}
