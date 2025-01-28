package orchestrator

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	Meter = otel.GetMeterProvider().Meter("orchestrator")
)

// Metrics for monitoring evaluation broker
var (
	EvalBrokerReady = telemetry.Must(Meter.Int64ObservableUpDownCounter(
		"eval_broker_ready",
		metric.WithDescription("Evaluations ready to be processed"),
	))

	EvalBrokerInflight = telemetry.Must(Meter.Int64ObservableUpDownCounter(
		"eval_broker_inflight",
		metric.WithDescription("Evaluations currently being processed"),
	))

	EvalBrokerPending = telemetry.Must(Meter.Int64ObservableUpDownCounter(
		"eval_broker_pending",
		metric.WithDescription("Duplicate evaluations for the same jobID pending for an active evaluation to finish"),
	))

	EvalBrokerWaiting = telemetry.Must(Meter.Int64ObservableUpDownCounter(
		"eval_broker_waiting",
		metric.WithDescription("Evaluations delayed and waiting to be processed"),
	))

	EvalBrokerCancelable = telemetry.Must(Meter.Int64ObservableUpDownCounter(
		"eval_broker_cancelable",
		metric.WithDescription("Duplicate evaluations for the same jobID that can be canceled"),
	))
)

// Message handler metrics
var (
	// Message processing metrics
	messageHandlerProcessDuration = telemetry.Must(Meter.Float64Histogram(
		"message.handler.process.duration",
		metric.WithDescription("Time taken to process a single message"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	messageHandlerProcessPartDuration = telemetry.Must(Meter.Float64Histogram(
		"message.handler.process.part.duration",
		metric.WithDescription("Time taken for sub-operations within message handling"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	messageHandlerProcessCount = telemetry.Must(Meter.Int64Counter(
		"message.handler.process.count",
		metric.WithDescription("Number of messages processed"),
		metric.WithUnit("1"),
	))
)

const (
	AttrEvalType    = "eval_type"
	AttrMessageType = "message_type"

	AttrPartBeginTx    = "begin_transaction"
	AttrPartGetJob     = "get_job"
	AttrPartCommitTx   = "commit_transaction"
	AttrPartUpdateExec = "update_execution"
	AttrPartCreateEval = "create_evaluation"

	AttrOutcomeKey     = "outcome"
	AttrOutcomeSuccess = "success"
	AttrOutcomeFailure = "failure"
)

func EvalTypeAttribute(evaluationType string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String(AttrEvalType, evaluationType))
}
