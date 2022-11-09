package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/joho/godotenv"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
)

const (
	TracerNameKey model.KeyString = "TracerName"
)

var tracer oteltrace.Tracer

type cleanupFn func() error
type cleanupTraceProviderFn func() error

// CleanupTracer should be called at the end of a node's execution to send all
// remaining traces to the exporter before the process ends.
var CleanupTracer cleanupFn
var CleanupTraceProviderFn cleanupTraceProviderFn

func init() { //nolint:gochecknoinits // use of init here is idomatic
	_, _ = NewTraceProvider()
}

// ----------------------------------------
// Tracer Setup and Teardown
// ----------------------------------------

func NewTraceProvider() (*sdktrace.TracerProvider, error) {
	_ = godotenv.Load() // Load environment variables from .env file - necessary here for dev keys

	tp, cleanup, err := hcTraceProvider()
	if err != nil {
		// don't error here because for CLI users they get a red message
		log.Trace().Msgf("error initializing http tracer: %v", err)
		log.Trace().Msg("failed to initialize http tracer, falling back to debug tracer")

		tp, cleanup, err = loggerTraceProvider()
		if err != nil {
			log.Error().Msgf("error initializing debug tracer: %v", err)
			log.Warn().Msg("failed to initialize debug tracer, will proceed without trace instrumentation")
			return tp, err // not fatal
		}
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	CleanupTraceProviderFn = cleanup

	tracer = tp.Tracer(version.TracerName())

	return tp, nil
}

func CleanupTraceProvider() error {
	return CleanupTraceProviderFn()
}

// ----------------------------------------
// Tracer helpers
// ----------------------------------------

func GetTracer() oteltrace.Tracer {
	return tracer
}

func GetTracerWithOpts(opts ...oteltrace.TracerOption) oteltrace.Tracer {
	tp := otel.GetTracerProvider()
	return tp.Tracer(version.TracerName(), opts...)
}

// ----------------------------------------
// Span helpers
// ----------------------------------------

func NewRootSpan(ctx context.Context, t oteltrace.Tracer, name string) (
	context.Context, oteltrace.Span) {
	// Always include environment info in spans:
	m0, _ := baggage.NewMember(string("environment"), GetEnvironment().String())
	b, _ := baggage.New(m0)
	ctx = baggage.ContextWithBaggage(ctx, b)

	return t.Start(ctx, name)
}

func GetSpanFromRequest(req *http.Request, name string) (context.Context, oteltrace.Span) {
	ctx := req.Context()
	_, span := tracer.Start(ctx, name)
	return ctx, span
}

func GetSpanFromContext(ctx context.Context) oteltrace.Span {
	return oteltrace.SpanFromContext(ctx)
}

// Span creates and starts a new span, and a context containing it.
// For more information see the otel.Tracer.Start(...) docs:
// https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer
// ctx: the context to use for the span
// tracerName: the name of the service that the span is for - will be prefixed with "tracer/".
// Will create a new one if one with the same name does not exist
// spanName: the name of the span, inside the service
// opts: additional options to configure the span from trace.SpanStartOption
func Span(ctx context.Context, tracerName, spanName string,
	opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	// Always include environment info in spans:
	opts = append(opts, oteltrace.WithAttributes(
		attribute.String("environment", GetEnvironment().String()),
	))

	spanName = fmt.Sprintf("service/%s", spanName)

	return GetTracer().Start(ctx, spanName, opts...)
}

// ----------------------------------------
// Providers
// ----------------------------------------

// httpProvider provides traces that are exported over GRPC to Honeycomb. It
// should be configured by setting the following environment variable:
//
//	export HONEYCOMB_KEY="<honeycomb api key>"
func hcTraceProvider() (*sdktrace.TracerProvider, cleanupTraceProviderFn, error) {
	honeycombDataset := os.Getenv("HONEYCOMB_DATASET")
	if honeycombDataset == "" {
		honeycombDataset = "bacalhau-unset-dataset"
	}
	log.Trace().Msgf("using honeycomb dataset: %s", honeycombDataset)

	honeycombKey := os.Getenv("HONEYCOMB_KEY")
	log.Trace().Msgf("using honeycomb key: %s", honeycombKey)

	if honeycombKey == "" {
		return nil, nil, fmt.Errorf(
			"error creating honeycomb exporter: please ensure that \"HONEYCOMB_KEY\" has been set")
	}

	exp, err := hcExporter(honeycombKey, honeycombDataset)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating honeycomb exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp), // TODO: use WithBatcher in prod
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("bacalhau"),
			),
		),
	)

	return tp, cleanupForTP(tp), nil
}

func loggerTraceProvider() (*sdktrace.TracerProvider, cleanupTraceProviderFn, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(jsonLogger()))
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp))

	return tp, cleanupForTP(tp), nil
}

// hcExporter returns a SpanExporter configured for Honeycomb.
func hcExporter(honeycombKey, honeycombDataset string) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
		otlptracegrpc.WithTLSCredentials(
			credentials.NewClientTLSFromCert(nil, "")),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    honeycombKey,
			"x-honeycomb-dataset": honeycombDataset,
		}),
	}

	// TODO: #580 Should this be cmd.Context()?
	return otlptrace.New(context.Background(),
		otlptracegrpc.NewClient(opts...))
}

// jsonLogger returns a writer than trace logs all JSON objects thrown at it.
func jsonLogger() io.Writer {
	r, w := io.Pipe()
	go func(r io.Reader) {
		d := json.NewDecoder(r)

		for {
			var data map[string]interface{}
			if err := d.Decode(&data); err != nil {
				if err == io.EOF {
					return
				}

				log.Trace().Msgf("error parsing json span: %v", err)
				continue
			}

			bs, err := json.Marshal(data)
			if err != nil {
				log.Trace().Msgf("error marshaling json span: %v", err)
				continue
			}

			log.Trace().Msg(string(bs))
		}
	}(r)

	return w
}

// cleanupFor returns a cleanup function that flushes remaining spans in
// memory to the exporter and releases any tracing resources.
//
//nolint:unparam // will add tracing
func cleanupForTP(tp *sdktrace.TracerProvider) cleanupTraceProviderFn {
	// TODO: #581 The below is wrong - we need to shut down the trace provider and take the context from the caller.
	return func() error {
		if err := tp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf(
				"error shutting down trace provider: %+v", err)
		}
		return nil
	}
}

// ----------------------------------------
// Baggage and Attribute helpers
// ----------------------------------------

func AddNodeIDToBaggage(ctx context.Context, nodeID string) context.Context {
	return addFieldToBaggage(ctx, model.TracerAttributeNameNodeID, nodeID)
}

func AddJobIDToBaggage(ctx context.Context, jobID string) context.Context {
	return addFieldToBaggage(ctx, model.TracerAttributeNameJobID, jobID)
}

func addFieldToBaggage(ctx context.Context, key, value string) context.Context {
	b := baggage.FromContext(ctx)
	m, err := baggage.NewMember(key, value)
	if err != nil {
		log.Warn().Msgf("failed to add key %s to baggage: %s", key, err)
	}

	b, err = b.SetMember(m)
	if err != nil {
		log.Warn().Msgf("failed to add baggage member to baggage: %s", err)
	}

	return baggage.ContextWithBaggage(ctx, b)
}

func AddJobIDFromBaggageToSpan(ctx context.Context, span oteltrace.Span) {
	AddAttributeToSpanFromBaggage(ctx, span, model.TracerAttributeNameJobID)
}

func AddNodeIDFromBaggageToSpan(ctx context.Context, span oteltrace.Span) {
	AddAttributeToSpanFromBaggage(ctx, span, model.TracerAttributeNameNodeID)
}

func AddAttributeToSpanFromBaggage(ctx context.Context, span oteltrace.Span, name string) {
	b := baggage.FromContext(ctx)
	log.Trace().Msgf("adding %s from baggage to span as attribute: %+v", name, b)
	m := b.Member(name)
	if m.Value() != "" {
		span.SetAttributes(attribute.String(name, m.Value()))
	} else {
		log.Trace().Msgf("no value found for baggage key %s", name)
		if log.Trace().Enabled() {
			debug.PrintStack()
		}
	}
}
