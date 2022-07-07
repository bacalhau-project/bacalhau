package system

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/system"
	_ "github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTracer(t *testing.T) {
	defer system.CleanupTracer()

	var sr SpanRecorder
	tp := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	tp.RegisterSpanProcessor(&sr)

	ctx := context.Background()
	ctx, span1 := system.Span(ctx, "service", "span1")
	ctx, span2 := system.Span(ctx, "service", "span2")
	span2.End()
	span1.End()

	assert.Len(t, sr.traces, 2)
	require.Equal(t, "service/span1", sr.traces[0].Name())
	require.Equal(t, "service/span2", sr.traces[1].Name())
}

// SpanRecorder is an implementation of sdktrace.SpanProcessor that records
// spans as they are created.
type SpanRecorder struct {
	traces []trace.ReadWriteSpan
}

func (sr *SpanRecorder) Shutdown(context.Context) error   { return nil }
func (sr *SpanRecorder) ForceFlush(context.Context) error { return nil }
func (sr *SpanRecorder) OnEnd(s sdktrace.ReadOnlySpan)    {}
func (sr *SpanRecorder) OnStart(ctx context.Context,
	span sdktrace.ReadWriteSpan) {

	sr.traces = append(sr.traces, span)
}
