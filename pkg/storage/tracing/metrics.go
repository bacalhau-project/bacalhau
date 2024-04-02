package tracing

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.GetMeterProvider().Meter("storage")

	jobStorageUploadDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"job_storage_upload_duration_milliseconds",
		metric.WithDescription("Duration of uploading job storage input the compute node in milliseconds."),
		metric.WithUnit("ms"),
	))

	jobStoragePrepareDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"job_storage_prepare_duration_milliseconds",
		metric.WithDescription("Duration of preparing job storage input the compute node in milliseconds."),
		metric.WithUnit("ms"),
	))

	jobStorageCleanupDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"job_storage_cleanup_duration_milliseconds",
		metric.WithDescription("Duration of job storage input cleanup the compute node in milliseconds."),
		metric.WithUnit("ms"),
	))
)
