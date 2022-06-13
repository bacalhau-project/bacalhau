package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationVersion = "0.0.1"

func init() {
	tp, err := debugProvider()
	if err != nil {
		log.Error().Msgf("error initialising tracer: %v", err)
		log.Warn().Msg("failed to initialise tracer, will proceed without trace instrumentation")
		return // not fatal
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
}

// Span creates and starts a new span, and a context containing it.
// For more information see the otel.Tracer.Start(...) docs:
//   https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer
func Span(ctx context.Context, svcName, spanName string,
	opts ...trace.SpanStartOption) (context.Context, trace.Span) {

	fqn := fmt.Sprintf("bacalhau/%s", svcName)
	return tracer(fqn).Start(ctx, spanName, opts...)
}

func tracer(svcName string) trace.Tracer {
	// TODO: set schema URL
	return otel.GetTracerProvider().Tracer(svcName,
		trace.WithInstrumentationVersion(instrumentationVersion))
}

func debugProvider() (trace.TracerProvider, error) {
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

//func honeycombProvider() (trace.TracerProvider, error) {
//	return nil, nil
//}

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
