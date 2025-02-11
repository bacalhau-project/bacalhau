package planner

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	Meter = otel.GetMeterProvider().Meter("planner")

	// Processing metrics
	processDuration = telemetry.Must(Meter.Float64Histogram(
		"planner.process.duration",
		metric.WithDescription("Time taken to process a single plan"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	processPartDuration = telemetry.Must(Meter.Float64Histogram(
		"planner.process.part.duration",
		metric.WithDescription("Time taken for sub-operations within a planner operation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	processCount = telemetry.Must(Meter.Int64Counter(
		"planner.process.count",
		metric.WithDescription("Number of plans processed"),
		metric.WithUnit("1"),
	))

	// State update metrics
	executionsCreated = telemetry.Must(Meter.Float64Histogram(
		"planner.executions.created",
		metric.WithDescription("Distribution of executions created per plan"),
		metric.WithUnit("1"),
	))

	executionsUpdated = telemetry.Must(Meter.Float64Histogram(
		"planner.executions.updated",
		metric.WithDescription("Distribution of executions updated per plan"),
		metric.WithUnit("1"),
	))

	jobsUpdated = telemetry.Must(Meter.Int64Counter(
		"planner.jobs.updated",
		metric.WithDescription("Number of jobs with state updates"),
		metric.WithUnit("1"),
	))

	evaluationsCreated = telemetry.Must(Meter.Float64Histogram(
		"planner.evaluations.created",
		metric.WithDescription("Distribution of evaluations created per plan"),
		metric.WithUnit("1"),
	))

	// History event metrics
	jobEventsAdded = telemetry.Must(Meter.Float64Histogram(
		"planner.events.job",
		metric.WithDescription("Distribution of job events added per plan"),
		metric.WithUnit("1"),
	))

	execEventsAdded = telemetry.Must(Meter.Float64Histogram(
		"planner.events.execution",
		metric.WithDescription("Distribution of execution events added per plan"),
		metric.WithUnit("1"),
	))
)

// Common attribute keys
const (
	AttrPlannerType = "planner_type"

	AttrOperationPartBeginTx    = "begin_transaction"
	AttrOperationPartCreateExec = "create_execution"
	AttrOperationPartUpdateExec = "update_execution"
	AttrOperationPartUpdateJob  = "update_job"
	AttrOperationPartCreateEval = "create_evaluation"
	AttrOperationPartAddEvents  = "add_events"

	AttrOutcomeKey     = "outcome"
	AttrOutcomeSuccess = "success"
	AttrOutcomeFailure = "failure"
)
