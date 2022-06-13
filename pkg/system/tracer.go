package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// CleanupTracer should be called at the end of a node's execution to send all
// remaining traces out the exporter before the process ends.
var CleanupTracer func() error

func init() {
	tp, err := httpProvider()
	if err != nil {
		log.Error().Msgf("error initialising http tracer: %v", err)
		log.Warn().Msg("failed to initialise http tracer, falling back to debug tracer")

		tp, err = debugProvider()
		if err != nil {
			log.Error().Msgf("error initialising debug tracer: %v", err)
			log.Warn().Msg("failed to initialise debug tracer, will proceed without trace instrumentation")
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

	CleanupTracer = func() error {
		if err := tp.ForceFlush(context.Background()); err != nil {
			log.Error().Msgf("error flushing remaining spans: %v", err)
		}

		return tp.Shutdown(context.Background())
	}
}

// Span creates and starts a new span, and a context containing it.
// For more information see the otel.Tracer.Start(...) docs:
//   https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer
func Span(ctx context.Context, svcName, spanName string,
	opts ...trace.SpanStartOption) (context.Context, trace.Span) {

	svc := fmt.Sprintf("bacalhau.org/%s", svcName)
	spn := fmt.Sprintf("%s/%s", svcName, spanName)
	return tracer(svc).Start(ctx, spn, opts...)
}

func tracer(svcName string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(svcName)
}

// debugProvider provides traces that are exported to a trace logger as JSON.
func debugProvider() (*sdktrace.TracerProvider, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(jsonLogger()))
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
	), nil
}

// httpProvider provides traces that are exported over HTTP to a server (such
// as Honeycomb) configured using the following environment variables:
//   export OTEL_EXPORTER_OTLP_ENDPOINT="https://api.honeycomb.io"
//   export OTEL_EXPORTER_OTLP_HEADERS="x-honeycomb-dataset=bacalhau,x-honeycomb-team=your-api-key"
//   export OTEL_SERVICE_NAME="your-honeycomb-service-name"
func httpProvider() (*sdktrace.TracerProvider, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_HEADERS") == "" ||
		os.Getenv("OTEL_SERVICE_NAME") == "" {

		return nil, fmt.Errorf("error creating http exporter: please ensure the \"OTEL_EXPORTER_OTLP_ENDPOINT\", \"OTEL_EXPORTER_OTLP_HEADERS\" and \"OTEL_SERVICE_NAME\" environment variables are set correctly")
	}

	cl := otlptracehttp.NewClient()
	exp, err := otlptrace.New(context.Background(), cl)
	if err != nil {
		return nil, fmt.Errorf("error creating http trace exporter: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("bacalhau.org"),
				semconv.ServiceVersionKey.String("0.0.1"),
			),
		),
	), nil

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
				log.Trace().Msgf("error marshalling json span: %v", err)
				continue
			}

			log.Trace().Msg(string(bs))
		}
	}(r)

	return w
}
