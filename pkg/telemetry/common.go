package telemetry

import (
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func SetupFromEnvs() {
	newTraceProvider()
	newMeterProvider()

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Err(err).Msg("Error occurred while handling spans")
	}))
}

// Cleanup flushes the remaining traces and metrics in memory to the exporter and releases any telemetry resources.
func Cleanup() error {
	tracingError := cleanupTraceProvider()
	meterError := cleanupMeterProvider()
	var err error
	if tracingError != nil || meterError != nil {
		err = errors.New("telemetry cleanup error")
		if tracingError != nil {
			err = errors.Wrap(err, "tracing cleanup error")
		}
		if meterError != nil {
			err = errors.Wrap(err, "meter cleanup error")
		}
	}
	return err
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	res, err := resource.Merge(
		resource.Environment(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("bacalhau"),
			semconv.ServiceVersionKey.String(version.GITVERSION),
		),
	)

	if err != nil {
		log.Error().Err(err).Msg("failed to create otel resource. Falling back to default resource config")
		res = resource.Default()
	}
	return res
}
