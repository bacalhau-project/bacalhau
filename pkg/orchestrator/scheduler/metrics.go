package scheduler

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	Meter = otel.GetMeterProvider().Meter("scheduler")

	// Processing metrics
	processDuration = telemetry.Must(Meter.Float64Histogram(
		"scheduler.process.duration",
		metric.WithDescription("Time taken to process a single evaluation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	processPartDuration = telemetry.Must(Meter.Float64Histogram(
		"scheduler.process.part.duration",
		metric.WithDescription("Time taken for sub-operations within a scheduler operation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	processCount = telemetry.Must(Meter.Int64Counter(
		"scheduler.process.count",
		metric.WithDescription("Number of evaluations processed"),
		metric.WithUnit("1"),
	))

	// Execution metrics
	executionsExisting = telemetry.Must(Meter.Float64Histogram(
		"scheduler.executions.existing",
		metric.WithDescription("Distribution of executions existing per job"),
		metric.WithUnit("1"),
	))

	executionsCreated = telemetry.Must(Meter.Float64Histogram(
		"scheduler.executions.created",
		metric.WithDescription("Distribution of executions created per evaluation"),
		metric.WithUnit("1"),
	))

	executionsCreatedTotal = telemetry.Must(Meter.Int64Counter(
		"scheduler.executions.created.total",
		metric.WithDescription("Total number of executions created"),
		metric.WithUnit("1"),
	))

	executionsLost = telemetry.Must(Meter.Float64Histogram(
		"scheduler.executions.lost",
		metric.WithDescription("Distribution of executions lost per evaluation"),
		metric.WithUnit("1"),
	))

	executionsLostTotal = telemetry.Must(Meter.Int64Counter(
		"scheduler.executions.lost.total",
		metric.WithDescription("Total number of executions lost"),
		metric.WithUnit("1"),
	))

	executionsTimedOut = telemetry.Must(Meter.Int64Counter(
		"scheduler.executions.timeout",
		metric.WithDescription("Number of executions that timed out"),
		metric.WithUnit("1"),
	))

	// Node metrics
	nodesMatched = telemetry.Must(Meter.Float64Histogram(
		"scheduler.nodes.matched",
		metric.WithDescription("Distribution of nodes matching job requirements"),
		metric.WithUnit("1"),
	))

	nodesRejected = telemetry.Must(Meter.Float64Histogram(
		"scheduler.nodes.rejected",
		metric.WithDescription("Distribution of nodes rejected for job requirements"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(telemetry.CountBuckets...),
	))

	// Retry metrics
	retriesExhausted = telemetry.Must(Meter.Int64Counter(
		"scheduler.retries.exhausted",
		metric.WithDescription("Number of jobs that exhausted retries"),
		metric.WithUnit("1"),
	))

	retriesAttempted = telemetry.Must(Meter.Int64Counter(
		"scheduler.retries.attempted",
		metric.WithDescription("Number of retries attempted"),
		metric.WithUnit("1"),
	))
)

// Common attribute keys
const (
	AttrJobType       = "job_type"
	AttrEvalType      = "eval_type"
	AttrSchedulerType = "scheduler_type"

	AttrOperationPartGetJob      = "get_job"
	AttrOperationPartGetExecs    = "get_executions"
	AttrOperationPartGetNodes    = "get_node_infos"
	AttrOperationPartMatchNodes  = "match_nodes"
	AttrOperationPartProcessPlan = "process_plan"

	AttrOutcomeKey              = attribute.Key("outcome")
	AttrOutcomeSuccess          = "success"
	AttrOutcomeFailure          = "failure"
	AttrOutcomeAlreadyTerminal  = "already_terminal"
	AttrOutcomeExhaustedRetries = "exhausted_retries"
	AttrOutcomeQueueing         = "queueing"
	AttrOutcomeTimeout          = "timeout"
	AttrOutcomeQueueTimeout     = "queue_timeout"
)
