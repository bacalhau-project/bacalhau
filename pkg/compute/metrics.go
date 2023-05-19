package compute

import (
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
)

// Metrics for monitoring compute nodes:
var (
	meter           = global.MeterProvider().Meter("compute")
	jobsReceived, _ = meter.Int64Counter(
		"jobs_received",
		instrument.WithDescription("Number of jobs received by the compute node"),
	)

	jobsAccepted, _ = meter.Int64Counter(
		"jobs_accepted",
		instrument.WithDescription("Number of jobs bid on and accepted by the compute node"),
	)

	jobsCompleted, _ = meter.Int64Counter(
		"jobs_completed",
		instrument.WithDescription("Number of jobs completed by the compute node."),
	)

	jobsFailed, _ = meter.Int64Counter(
		"jobs_failed",
		instrument.WithDescription("Number of jobs failed by the compute node."),
	)
)
