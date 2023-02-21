package system

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// A job event handler that adds lifecycle events to the job tracing span, both for events consumed and events published.
type JobLifecycleEventHandler struct {
	nodeID string
}

func NewJobLifecycleEventHandler(nodeID string) *JobLifecycleEventHandler {
	return &JobLifecycleEventHandler{
		nodeID: nodeID,
	}
}

func (t *JobLifecycleEventHandler) HandleConsumedJobEvent(ctx context.Context, event model.JobEvent) error {
	return t.addJobLifecycleEvent(ctx, event.JobID, fmt.Sprintf("read_%s", event.EventName))
}

func (t *JobLifecycleEventHandler) HandlePublishedJobEvent(ctx context.Context, event model.JobEvent) error {
	return t.addJobLifecycleEvent(ctx, event.JobID, fmt.Sprintf("write_%s", event.EventName))
}

func (t *JobLifecycleEventHandler) addJobLifecycleEvent(ctx context.Context, jobID, eventName string) error {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(eventName,
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, t.nodeID),
			attribute.String(model.TracerAttributeNameJobID, jobID),
		),
	)
	return nil
}
