package tracing

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.GetMeterProvider().Meter("publisher")

	jobPublishDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"job_publish_duration_milliseconds",
		metric.WithDescription("Duration of publishing a job on the compute node in milliseconds."),
		metric.WithUnit("ms"),
	))
)
