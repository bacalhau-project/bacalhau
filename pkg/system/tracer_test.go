//go:build unit || !integration

package system

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTracer(t *testing.T) {
	t.Cleanup(func() {
		assert.NoError(t, CleanupTraceProvider())
	})

	var sr SpanRecorder
	tp := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	tp.RegisterSpanProcessor(&sr)

	ctx := context.Background()
	ctx, span1 := Span(ctx, "service", "span1")
	ctx, span2 := Span(ctx, "service", "span2") //lint:ignore SA4006 ok to have extra assignment
	span2.End()
	span1.End()

	require.Len(t, sr.traces, 2)
	require.Equal(t, "service/span1", sr.traces[0].Name())
	require.Equal(t, "service/span2", sr.traces[1].Name())
}

// SpanRecorder is an implementation of sdktrace.SpanProcessor that records
// spans as they are created.
type SpanRecorder struct {
	traces []sdktrace.ReadWriteSpan
}

func (sr *SpanRecorder) Shutdown(context.Context) error   { return nil }
func (sr *SpanRecorder) ForceFlush(context.Context) error { return nil }
func (sr *SpanRecorder) OnEnd(sdktrace.ReadOnlySpan)      {}
func (sr *SpanRecorder) OnStart(_ context.Context,
	span sdktrace.ReadWriteSpan) {

	sr.traces = append(sr.traces, span)
}
