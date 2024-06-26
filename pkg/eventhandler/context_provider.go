package eventhandler

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

// Interface for a context provider that can be used to generate a context to be used to handle
// job events.
type ContextProvider interface {
	GetContext(ctx context.Context, jobID string) context.Context
}

// TracerContextProvider is a context provider that generates a context along with tracing information.
// It also implements JobEventHandler to end the local lifecycle context for a job when it is completed.
type TracerContextProvider struct {
	nodeID          string
	jobNodeContexts map[string]context.Context // per-node job lifecycle
	contextMutex    sync.RWMutex
}

func NewTracerContextProvider(nodeID string) *TracerContextProvider {
	return &TracerContextProvider{
		nodeID:          nodeID,
		jobNodeContexts: make(map[string]context.Context),
	}
}

func (t *TracerContextProvider) GetContext(ctx context.Context, jobID string) context.Context {
	t.contextMutex.Lock()
	defer t.contextMutex.Unlock()

	jobCtx, _ := system.Span(ctx, "pkg/eventhandler/JobEventHandler.HandleJobEvent",
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
		oteltrace.WithAttributes(
			attribute.String(telemetry.TracerAttributeNameNodeID, t.nodeID),
			attribute.String(telemetry.TracerAttributeNameJobID, jobID),
		),
	)

	// keep the latest context to clean it up during shutdown if necessary
	t.jobNodeContexts[jobID] = jobCtx
	return jobCtx
}

func (t *TracerContextProvider) HandleJobEvent(ctx context.Context, event models.JobEvent) error {
	// If the event is known to be terminal, end the local lifecycle context:
	if event.EventName.IsTerminal() {
		t.endJobNodeContext(ctx, event.JobID)
	}

	return nil
}

func (t *TracerContextProvider) Shutdown() error {
	t.contextMutex.RLock()
	defer t.contextMutex.RUnlock()

	for _, ctx := range t.jobNodeContexts {
		oteltrace.SpanFromContext(ctx).End()
	}

	// clear the maps
	t.jobNodeContexts = make(map[string]context.Context)

	return nil
}

// endJobNodeContext ends the local lifecycle context for a job.
func (t *TracerContextProvider) endJobNodeContext(ctx context.Context, jobID string) {
	oteltrace.SpanFromContext(ctx).End()
	t.contextMutex.Lock()
	defer t.contextMutex.Unlock()
	delete(t.jobNodeContexts, jobID)
}
