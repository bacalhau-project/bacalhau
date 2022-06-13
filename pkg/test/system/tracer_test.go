package system

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/system"
	_ "github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTracer(t *testing.T) {
	var sp SpanProcessor
	tp := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	tp.RegisterSpanProcessor(&sp)

	ctx := context.Background()
	ctx, span1 := system.Span(ctx, "service", "span1")
	ctx, span2 := system.Span(ctx, "service", "span2")
	span2.End()
	span1.End()

	assert.Len(t, sp.names, 2)
	assert.Equal(t, "service/span1", sp.names[0])
	assert.Equal(t, "service/span2", sp.names[1])
}

// SpanProcessor is an implementation of sdktrace.SpanProcessor that records
// the order in which spans are created for testing.
type SpanProcessor struct {
	names []string
}

func (sp *SpanProcessor) Shutdown(context.Context) error   { return nil }
func (sp *SpanProcessor) ForceFlush(context.Context) error { return nil }
func (sp *SpanProcessor) OnEnd(s sdktrace.ReadOnlySpan)    {}
func (sp *SpanProcessor) OnStart(ctx context.Context,
	span sdktrace.ReadWriteSpan) {

	sp.names = append(sp.names, span.Name())
}
