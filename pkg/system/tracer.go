package system

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel/propagation"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// ----------------------------------------
// Tracer helpers
// ----------------------------------------

func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer("bacalhau")
}

// ----------------------------------------
// Span helpers
// ----------------------------------------

func NewSpan(ctx context.Context, t trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	for _, attributeName := range []string{telemetry.TracerAttributeNameJobID, telemetry.TracerAttributeNameNodeID} {
		if v := baggage.FromContext(ctx).Member(attributeName).Value(); v != "" {
			opts = append(opts, trace.WithAttributes(
				attribute.String(attributeName, v),
			))
		}
	}
	opts = append(opts, trace.WithAttributes(
		attribute.String("environment", GetEnvironment().String()),
	))

	return t.Start(ctx, name, opts...)
}

func NewRootSpan(ctx context.Context, t trace.Tracer, name string) (context.Context, trace.Span) {
	// Always include environment info in spans:
	environment := GetEnvironment().String()
	m0, _ := baggage.NewMember("environment", environment)
	b, _ := baggage.New(m0)
	ctx = baggage.ContextWithBaggage(ctx, b)

	return t.Start(ctx, name, trace.WithAttributes(
		attribute.String("environment", environment),
	))
}

// Span creates and starts a new span, and a context containing it.
// For more information see the otel.Tracer.Start(...) docs:
// https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer
// ctx: the context to use for the span
// tracerName: the name of the service that the span is for - will be prefixed with "tracer/".
// Will create a new one if one with the same name does not exist
// spanName: the name of the span, inside the service
// opts: additional options to configure the span from trace.SpanStartOption
func Span(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Always include environment info in spans:
	opts = append(opts, trace.WithAttributes(
		attribute.String("environment", GetEnvironment().String()),
	))

	return GetTracer().Start(ctx, spanName, opts...)
}

// ----------------------------------------
// Baggage and Attribute helpers
// ----------------------------------------

func AddNodeIDToBaggage(ctx context.Context, nodeID string) context.Context {
	return addFieldToBaggage(ctx, telemetry.TracerAttributeNameNodeID, nodeID)
}

func AddJobIDToBaggage(ctx context.Context, jobID string) context.Context {
	return addFieldToBaggage(ctx, telemetry.TracerAttributeNameJobID, jobID)
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

// ----------------------------------------
// Propagation helpers
// ----------------------------------------

// injectContext serializes the trace context from the provided context.Context
func injectContext(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier
}

// extractSpanContext deserializes the trace context and returns a SpanContext
func extractSpanContext(metadata map[string]string) trace.SpanContext {
	carrier := propagation.MapCarrier(metadata)
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	return trace.SpanContextFromContext(ctx)
}

// InjectJobContext injects the trace context into a job's metadata
func InjectJobContext(ctx context.Context, job *models.Job) {
	carrier := injectContext(ctx)
	jobContext, err := json.Marshal(carrier)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to inject job tracing context")
		return
	}
	if job.Meta == nil {
		job.Meta = make(map[string]string)
	}
	job.Meta[models.MetaTraceContext] = string(jobContext)
}

// ExtractJobSpanContext extracts the trace context from a job's metadata and returns a SpanContext
func ExtractJobSpanContext(job models.Job) trace.SpanContext {
	jobContextJSON, ok := job.Meta[models.MetaTraceContext]
	if !ok {
		return trace.SpanContext{}
	}
	var jobContext map[string]string
	if err := json.Unmarshal([]byte(jobContextJSON), &jobContext); err != nil {
		log.Warn().Err(err).Msgf("failed to extract job tracing context")
		return trace.SpanContext{}
	}
	return extractSpanContext(jobContext)
}
