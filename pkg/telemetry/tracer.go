package telemetry

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	oteltrace "go.opentelemetry.io/otel/trace"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
)

// ----------------------------------------
// Tracer helpers
// ----------------------------------------

func GetTracer() oteltrace.Tracer {
	return otel.GetTracerProvider().Tracer("bacalhau")
}

// ----------------------------------------
// Span helpers
// ----------------------------------------

func NewSpan(ctx context.Context, t oteltrace.Tracer, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	for _, attributeName := range []string{TracerAttributeNameJobID, TracerAttributeNameNodeID} {
		if v := baggage.FromContext(ctx).Member(attributeName).Value(); v != "" {
			opts = append(opts, oteltrace.WithAttributes(
				attribute.String(attributeName, v),
			))
		}
	}
	return t.Start(ctx, name, opts...)
}

func NewRootSpan(ctx context.Context, t oteltrace.Tracer, name string) (context.Context, oteltrace.Span) {
	return t.Start(ctx, name)
}

// Span creates and starts a new span, and a context containing it.
// For more information see the otel.Tracer.Start(...) docs:
// https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer
// ctx: the context to use for the span
// tracerName: the name of the service that the span is for - will be prefixed with "tracer/".
// Will create a new one if one with the same name does not exist
// spanName: the name of the span, inside the service
// opts: additional options to configure the span from trace.SpanStartOption
func Span(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return GetTracer().Start(ctx, spanName, opts...)
}

// ----------------------------------------
// Baggage and Attribute helpers
// ----------------------------------------

func AddNodeIDToBaggage(ctx context.Context, nodeID string) context.Context {
	return addFieldToBaggage(ctx, TracerAttributeNameNodeID, nodeID)
}

func AddJobIDToBaggage(ctx context.Context, jobID string) context.Context {
	return addFieldToBaggage(ctx, TracerAttributeNameJobID, jobID)
}

func addFieldToBaggage(ctx context.Context, key, value string) context.Context {
	b := baggage.FromContext(ctx)
	m, err := baggage.NewMember(key, value)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to add key %s to baggage: %s", key, err)
	}

	b, err = b.SetMember(m)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to add baggage member to baggage: %s", err)
	}

	return baggage.ContextWithBaggage(ctx, b)
}
