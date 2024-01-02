package orchestrator

import (
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	Meter = otel.GetMeterProvider().Meter("orchestrator")
)

// Metrics for monitoring worker
var (
	WorkerDequeueFaults = telemetry.Must(Meter.Int64Counter(
		"worker_dequeue_faults",
		metric.WithDescription("Number of times a worker failed to dequeue an evaluation"),
	))

	WorkerProcessFaults = telemetry.Must(Meter.Int64Counter(
		"worker_process_faults",
		metric.WithDescription("Number of times a worker failed to process an evaluation"),
	))

	WorkerAckFaults = telemetry.Must(Meter.Int64Counter(
		"worker_ack_faults",
		metric.WithDescription("Number of times a worker failed to ack an evaluation back to the broker"),
	))

	WorkerNackFaults = telemetry.Must(Meter.Int64Counter(
		"worker_nack_faults",
		metric.WithDescription("Number of times a worker failed to nack an evaluation back to the broker"),
	))
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

func EvalTypeAttribute(evaluationType string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("eval_type", evaluationType))
}
