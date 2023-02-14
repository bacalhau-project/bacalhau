package eventhandler

import (
	"context"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Interface for a context provider that can be used to generate a context to be used to handle
// job events.
type ContextProvider interface {
	GetContext(ctx context.Context, jobID string) context.Context
}

// NoopContextProvider is a context provider that does not generate a new context, and
// simply returns the	ctx passed in.
type NoopContextProvider struct{}

func NewNoopContextProvider() *NoopContextProvider {
	return &NoopContextProvider{}
}

func (t *NoopContextProvider) GetContext(ctx context.Context, _ string) context.Context {
	return ctx
}

// TracerContextProvider is a context provider that generates a context along with tracing information.
// It also implements JobEventHandler to end the local lifecycle context for a job when it is completed.
type TracerContextProvider struct {
	nodeID          string
	jobNodeContexts map[string]context.Context // per-node job lifecycle
	contextMutex    sync.RWMutex
}

func NewTracerContextProvider(nodeID string) *TracerContextProvider {
	tracer := &TracerContextProvider{
		nodeID:          nodeID,
		jobNodeContexts: make(map[string]context.Context),
	}

	tracer.contextMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "Tracer.contextMutex",
	})
	return tracer
}

func (t *TracerContextProvider) GetContext(ctx context.Context, jobID string) context.Context {
	t.contextMutex.Lock()
	defer t.contextMutex.Unlock()

	jobCtx, _ := system.Span(ctx, "pkg/eventhandler/JobEventHandler.HandleJobEvent",
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
		oteltrace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, t.nodeID),
			attribute.String(model.TracerAttributeNameJobID, jobID),
		),
	)

	// keep the latest context to clean it up during shutdown if necessary
	t.jobNodeContexts[jobID] = jobCtx
	return jobCtx
}

func (t *TracerContextProvider) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	// If the event is known to be ignorable, end the local lifecycle context:
	if event.EventName.IsIgnorable() {
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
