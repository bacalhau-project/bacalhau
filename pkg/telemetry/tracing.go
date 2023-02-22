package telemetry

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ----------------------------------------
// Tracer Setup and Teardown
// ----------------------------------------
func newTraceProvider() {
	// The context passed in to the exporter is only passed to the client and used when connecting to the endpoint
	ctx := context.Background()

	if !isTracingEnabled() {
		log.Ctx(ctx).Debug().Msgf("OLTP tracing endpoints are not defined. No traces will be exported")
		return
	}

	client, err := getTraceClient()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to initialize OLTP trace client")
		return
	}

	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to initialize OLTP trace exporter")
		return
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(newResource()),
	)

	// set the global trace provider
	otel.SetTracerProvider(loggingTracerProvider{tp})

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
}

func getTraceClient() (client otlptrace.Client, err error) {
	protocol := otlpProtocolHTTP
	if v := os.Getenv(otlpProtocol); v != "" {
		protocol = v
	}
	if v := os.Getenv(otlpTracesProtocol); v != "" {
		protocol = v
	}
	switch protocol {
	case otlpProtocolHTTP:
		client = otlptracehttp.NewClient()
	case otlpProtocolGrpc:
		client = otlptracegrpc.NewClient()
	default:
		err = fmt.Errorf("unknown or unsupported OLTP protocol: %s. No traces will be exported", protocol)
	}
	return
}

func isTracingEnabled() bool {
	_, endpointDefined := os.LookupEnv(otlpEndpoint)
	_, tracingEndpointDefined := os.LookupEnv(otlpTracesEndpoint)
	return endpointDefined || tracingEndpointDefined
}

func cleanupTraceProvider() error {
	tracer, ok := otel.GetTracerProvider().(shutdownTracerProvider)
	if ok {
		return tracer.Shutdown(context.Background())
	}
	return nil
}

type shutdownTracerProvider interface {
	oteltrace.TracerProvider
	Shutdown(ctx context.Context) error
}
