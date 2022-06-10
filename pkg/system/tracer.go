package system

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TODO: hook up exporter for honeycomb, currently traces go nowhere
// https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlptrace
var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/filecoin-project/bacalhau")
}

// Span creates a span and a context containing the newly-created span.
// For more information see the otel.Tracer.Start(...) docs:
//   https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer
func Span(ctx context.Context, spanName string,
	opts ...trace.SpanStartOption) (context.Context, trace.Span) {

	return tracer.Start(ctx, spanName, opts...)
}
