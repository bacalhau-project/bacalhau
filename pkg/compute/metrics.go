package compute

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics for monitoring compute nodes:
var (
	meter        = otel.GetMeterProvider().Meter("compute")
	jobsReceived = lo.Must(meter.Int64Counter(
		"jobs_received",
		metric.WithDescription("Number of jobs received by the compute node"),
	))

	jobsAccepted = lo.Must(meter.Int64Counter(
		"jobs_accepted",
		metric.WithDescription("Number of jobs bid on and accepted by the compute node"),
	))

	jobsCompleted = lo.Must(meter.Int64Counter(
		"jobs_completed",
		metric.WithDescription("Number of jobs completed by the compute node."),
	))

	jobsFailed = lo.Must(meter.Int64Counter(
		"jobs_failed",
		metric.WithDescription("Number of jobs failed by the compute node."),
	))

	jobDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"job_duration_milliseconds",
		metric.WithDescription("Duration of a job on the compute node in milliseconds."),
		metric.WithUnit("ms"),
	))
)
