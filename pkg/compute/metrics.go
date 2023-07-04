package compute

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics for monitoring compute nodes:
var (
	meter           = otel.GetMeterProvider().Meter("compute")
	jobsReceived, _ = meter.Int64Counter(
		"jobs_received",
		metric.WithDescription("Number of jobs received by the compute node"),
	)

	jobsAccepted, _ = meter.Int64Counter(
		"jobs_accepted",
		metric.WithDescription("Number of jobs bid on and accepted by the compute node"),
	)

	jobsCompleted, _ = meter.Int64Counter(
		"jobs_completed",
		metric.WithDescription("Number of jobs completed by the compute node."),
	)

	jobsFailed, _ = meter.Int64Counter(
		"jobs_failed",
		metric.WithDescription("Number of jobs failed by the compute node."),
	)
)
