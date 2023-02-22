package telemetry

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric/global"
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

	// reader that also bridges opencensus metrics to capture libp2p metrics
	reader := sdkmetric.NewPeriodicReader(exp)
	reader.RegisterProducer(opencensus.NewMetricProducer())

	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(newResource()),
		sdkmetric.WithReader(reader),
	)

	global.SetMeterProvider(meterProvider)
}

func isMetricsEnabled() bool {
	_, endpointDefined := os.LookupEnv(otlpEndpoint)
	_, metricsEndpointDefined := os.LookupEnv(otlpMetricsEndpoint)
	return endpointDefined || metricsEndpointDefined
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
