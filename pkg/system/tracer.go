package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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

type cleanupFn func() error
type cleanupTraceProviderFn func(context.Context) error

// CleanupTracer should be called at the end of a node's execution to send all
// remaining traces to the exporter before the process ends.
var CleanupTracer cleanupFn
var CleanupTraceProviderFn cleanupTraceProviderFn

func init() { //nolint:gochecknoinits // use of init here is idomatic
	_ = godotenv.Load() // Load environment variables from .env file - necessary here for dev keys

	tp, cleanup, err := hcProvider()
	if err != nil {
		// don't error here because for CLI users they get a red message
		log.Debug().Msgf("error initializing http tracer: %v", err)
		log.Debug().Msg("failed to initialize http tracer, falling back to debug tracer")

		tp, cleanup, err = loggerProvider()
		if err != nil {
			log.Error().Msgf("error initializing debug tracer: %v", err)
			log.Warn().Msg("failed to initialize debug tracer, will proceed without trace instrumentation")
			return // not fatal
		}
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	CleanupTracer = cleanup
}

func NewTraceProvider() (*sdktrace.TracerProvider, error) {
	_ = godotenv.Load() // Load environment variables from .env file - necessary here for dev keys

	tp, cleanup, err := hcTraceProvider()
	if err != nil {
		// don't error here because for CLI users they get a red message
		log.Debug().Msgf("error initializing http tracer: %v", err)
		log.Debug().Msg("failed to initialize http tracer, falling back to debug tracer")

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

	return tp, nil
}

func CleanupTraceProvider(tp *sdktrace.TracerProvider, ctx context.Context) error {
	return CleanupTraceProviderFn(ctx)
}

func GetTracer(opts ...oteltrace.TracerOption) oteltrace.Tracer {
	tp := otel.GetTracerProvider()
	return tp.Tracer(version.TracerName(), opts...)
}

func GetTracerFromRequest(req *http.Request) (context.Context, oteltrace.Tracer) {
	ctx := req.Context()
	span := oteltrace.SpanFromContext(ctx)
	defer span.End()
	t := span.TracerProvider().Tracer(version.TracerName())
	return ctx, t
}

//nolint:lll // long line is ok here
func NewRootSpan(ctx context.Context, t oteltrace.Tracer, name string) (context.Context, oteltrace.Span) {
	// Always include environment info in spans:
	m0, _ := baggage.NewMember(string("environment"), GetEnvironment().String())
	b, _ := baggage.New(m0)
	ctx = baggage.ContextWithBaggage(ctx, b)

	return t.Start(ctx, name)
}

func GetSpanFromContext(ctx context.Context) oteltrace.Span {
	return oteltrace.SpanFromContext(ctx)
}

// Span creates and starts a new span, and a context containing it.
// For more information see the otel.Tracer.Start(...) docs:
// https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer
// ctx: the context to use for the span
// tracerName: the name of the service that the span is for - will be prefixed with "tracer/".
//		Will create a new one if one with the same name does not exist
// spanName: the name of the span, inside the service
// opts: additional options to configure the span from trace.SpanStartOption
func Span(ctx context.Context, tracerName, spanName string,
	opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	// Always include environment info in spans:
	opts = append(opts, oteltrace.WithAttributes(
		attribute.String("environment", GetEnvironment().String()),
	))

	spanName = fmt.Sprintf("service/%s", spanName)

	return Tracer(tracerName).Start(ctx, spanName, opts...)
}

func Tracer(tracerName string) oteltrace.Tracer {
	return otel.GetTracerProvider().Tracer(tracerName)
}

// loggerProvider provides traces that are exported to a trace logger as JSON.
func loggerProvider() (*sdktrace.TracerProvider, cleanupFn, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(jsonLogger()))
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp))

	return tp, cleanupFor("logger", tp, exp), nil
}

// httpProvider provides traces that are exported over GRPC to Honeycomb. It
// should be configured by setting the following environment variable:
//
//	export HONEYCOMB_KEY="<honeycomb api key>"
func hcProvider() (*sdktrace.TracerProvider, cleanupFn, error) {
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

	return tp, cleanupFor("honeycomb", tp, exp), nil
}

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
//nolint:unparam // will add tracing
func cleanupForTP(tp *sdktrace.TracerProvider) cleanupTraceProviderFn {
	return func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf(
				"error shutting down trace provider: %+v", err)
		}

		return nil
	}
}

// cleanupFor returns a cleanup function that flushes remaining spans in
// memory to the exporter and releases any tracing resources.
// TODO: #288 Use trace to close out span
//
//nolint:unparam // will add tracing
func cleanupFor(name string, tp *sdktrace.TracerProvider, exp sdktrace.SpanExporter) cleanupFn {
	return func() error {
		if err := tp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf(
				"error shutting down %s provider: %w", name, err)
		}

		return nil
	}
}
