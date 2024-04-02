package telemetry

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var meterProvider *sdkmetric.MeterProvider

func newMeterProvider() {
	// The context passed in to the exporter is only passed to the client and used when connecting to the endpoint
	ctx := context.Background()

	if !isMetricsEnabled() {
		log.Ctx(ctx).Debug().Msgf("OLTP metrics endpoints are not defined. No metrics will be exported")
		return
	}

	exp, err := getMetricsClient(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to initialize OLTP metric exporter")
		return
	}

	reader := sdkmetric.NewPeriodicReader(exp)

	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(newResource()),
		sdkmetric.WithReader(reader),
	)

	otel.SetMeterProvider(meterProvider)
}

func isMetricsEnabled() bool {
	if v, ok := os.LookupEnv(disableTracing); ok && v == "1" {
		return false
	}
	if _, ok := os.LookupEnv(otlpEndpoint); ok {
		return true
	}
	if _, ok := os.LookupEnv(otlpMetricsEndpoint); ok {
		return true
	}

	return false
}

func getMetricsClient(ctx context.Context) (client sdkmetric.Exporter, err error) {
	protocol := otlpProtocolHTTP
	if v := os.Getenv(otlpProtocol); v != "" {
		protocol = v
	}
	if v := os.Getenv(otlpMetricsProtocol); v != "" {
		protocol = v
	}
	switch protocol {
	case otlpProtocolHTTP:
		client, err = otlpmetrichttp.New(ctx)
	case otlpProtocolGrpc:
		client, err = otlpmetricgrpc.New(ctx)
	default:
		err = fmt.Errorf("unknown or unsupported OLTP protocol: %s. No metrics will be exported", protocol)
	}
	return
}

func cleanupMeterProvider() (err error) {
	if meterProvider != nil {
		ctx := context.Background()
		err = meterProvider.ForceFlush(ctx)
		if err != nil {
			err = meterProvider.Shutdown(ctx)
		}
	}
	return err
}
