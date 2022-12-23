package eventhandler

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// A local event handler is a component that is notified of local events that happen within the node.
type LocalEventHandler interface {
	HandleLocalEvent(ctx context.Context, event model.JobLocalEvent) error
}

// A job event handler is a component that is notified of events related to jobs.
type JobEventHandler interface {
	HandleJobEvent(ctx context.Context, event model.JobEvent) error
}

// A node event handler is a component that is notified of events related to nodes.
type NodeEventHandler interface {
	HandleNodeEvent(ctx context.Context, event model.NodeEvent) error
}

// function that implements the LocalEventHandler interface
type LocalEventHandlerFunc func(ctx context.Context, event model.JobLocalEvent) error

func (f LocalEventHandlerFunc) HandleLocalEvent(ctx context.Context, event model.JobLocalEvent) error {
	return f(ctx, event)
}

// function that implements the JobEventHandler interface
type JobEventHandlerFunc func(ctx context.Context, event model.JobEvent) error

func (f JobEventHandlerFunc) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	return f(ctx, event)
}

// function that implements the NodeEventHandler interface
type NodeEventHandlerFunc func(ctx context.Context, event model.NodeEvent) error

func (f NodeEventHandlerFunc) HandleNodeEvent(ctx context.Context, event model.NodeEvent) error {
	return f(ctx, event)
}
