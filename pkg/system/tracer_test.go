//go:build unit || !integration

package system

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel"
)

func TestTracer(t *testing.T) {
	t.Cleanup(func() {
		assert.NoError(t, telemetry.Cleanup())
	})

	var sr SpanRecorder
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tp.RegisterSpanProcessor(&sr)

	ctx := context.Background()
	ctx, span1 := Span(ctx, "span1")
	_, span2 := Span(ctx, "span2")
	span2.End()
	span1.End()

	require.Len(t, sr.traces, 2)
	require.Equal(t, "span1", sr.traces[0].Name())
	require.Equal(t, "span2", sr.traces[1].Name())
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
