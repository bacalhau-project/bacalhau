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
	"github.com/spf13/viper"

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

var tracer oteltrace.Tracer

func init() { //nolint:gochecknoinits // use of init here is idomatic
	newTraceProvider()
}

// ----------------------------------------
// Tracer Setup and Teardown
// ----------------------------------------

func newTraceProvider() {
	_ = godotenv.Load() // Load environment variables from .env file - necessary here for dev keys

	setViperFromLegacyHoneycombValues()
	tp, err := otelTraceProvider()
	if err != nil {
		// don't error here because for CLI users they get a red message
		log.Trace().Err(err).Msg("failed to initialize tracer, falling back to logging tracer")

		tp, err = loggerTraceProvider()
		if err != nil {
			// need to panic now with a nice message otherwise we'd throw a nil pointer dereference panic when we access tracer
			panic(fmt.Errorf("failed to initialize debug tracer: %w", err))
		}
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		// Block this common message from spamming the logs. It seems to be coming from
		// go.opentelemetry.io/otel/exporters/otlp/internal PartialSuccess
		// Should be fixed by https://github.com/open-telemetry/opentelemetry-go/issues/3432 (v1.12+)
		if err.Error() == "OTLP partial success: empty message (0 spans rejected)" {
			return
		}
		log.Err(err).Msg("Error occurred while handling spans")
	}))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tracer = tp.Tracer(version.TracerName())
}

// CleanupTraceProvider flushes the remaining spans in memory to the exporter and releases any tracing resources.
func CleanupTraceProvider() error {
	type shutdown interface {
		oteltrace.TracerProvider
		Shutdown(ctx context.Context) error
	}
	return otel.GetTracerProvider().(shutdown).Shutdown(context.Background())
}

// ----------------------------------------
// Tracer helpers
// ----------------------------------------

func GetTracer() oteltrace.Tracer {
	return tracer
}

// ----------------------------------------
// Span helpers
// ----------------------------------------

func NewRootSpan(ctx context.Context, t oteltrace.Tracer, name string) (context.Context, oteltrace.Span) {
	// Always include environment info in spans:
	m0, _ := baggage.NewMember("environment", GetEnvironment().String())
	b, _ := baggage.New(m0)
	ctx = baggage.ContextWithBaggage(ctx, b)

	return t.Start(ctx, name)
}

func GetSpanFromRequest(req *http.Request, name string) (context.Context, oteltrace.Span) {
	ctx := req.Context()
	ctx, span := tracer.Start(ctx, name)
	return ctx, span
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

func setViperFromLegacyHoneycombValues() {
	if viper.IsSet("trace_endpoint") {
		return
	}

	honeycombKey := os.Getenv("HONEYCOMB_KEY")
	if honeycombKey == "" {
		return
	}

	honeycombDataset := os.Getenv("HONEYCOMB_DATASET")
	if honeycombDataset == "" {
		honeycombDataset = "bacalhau-unset-dataset"
	}

	viper.Set("trace_endpoint", "api.honeycomb.io:443")
	viper.Set("trace_insecure", false)
	viper.Set("trace_headers", map[string]string{
		"x-honeycomb-team":    honeycombKey,
		"x-honeycomb-dataset": honeycombDataset,
	})
}

func otelTraceProvider() (*sdktrace.TracerProvider, error) {
	if !viper.IsSet("trace_endpoint") {
		return nil, fmt.Errorf("no trace endpoint configured")
	}

	options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(viper.GetString("trace_endpoint"))}

	if viper.IsSet("trace_insecure") && viper.GetBool("trace_insecure") {
		options = append(options, otlptracegrpc.WithInsecure())
	} else {
		options = append(options,
			otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	if viper.IsSet("trace_headers") {
		options = append(options, otlptracegrpc.WithHeaders(viper.GetStringMapString("trace_headers")))
	}

	// The context passed in to the exporter is only passed to the client and used when connecting to the endpoint
	exp, err := otlptrace.New(context.Background(), otlptracegrpc.NewClient(options...))
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp), // TODO: use WithBatcher in prod
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("bacalhau"),
			),
		),
	), nil
}

func loggerTraceProvider() (*sdktrace.TracerProvider, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(jsonLogger()))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp))

	return tp, nil
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

			bs, err := model.JSONMarshalWithMax(data)
			if err != nil {
				log.Trace().Msgf("error marshaling json span: %v", err)
				continue
			}

			log.Trace().Msg(string(bs))
		}
	}(r)

	return w
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
