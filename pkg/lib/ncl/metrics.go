package ncl

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	Meter = otel.GetMeterProvider().Meter("ncl")

	// Publish operation metrics
	publishDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.duration",
		metric.WithDescription("Time taken for publish operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	publishPartDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.part.duration",
		metric.WithDescription("Time taken for sub-operations within a publish"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	publishCount = telemetry.Must(Meter.Int64Counter(
		"ncl.publisher.message.count",
		metric.WithDescription("Number of messages published"),
		metric.WithUnit("1"),
	))

	publishBytes = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.message.bytes",
		metric.WithDescription("Size of published messages"),
		metric.WithUnit("By"),
	))

	// Request operation metrics
	requesterDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.requester.duration",
		metric.WithDescription("Time taken for request operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	requesterPartDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.requester.part.duration",
		metric.WithDescription("Time taken for sub-operations within a request"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	requesterCount = telemetry.Must(Meter.Int64Counter(
		"ncl.requester.request.count",
		metric.WithDescription("Number of requests made"),
		metric.WithUnit("1"),
	))

	requesterBytes = telemetry.Must(Meter.Float64Histogram(
		"ncl.requester.request.bytes",
		metric.WithDescription("Size of request messages"),
		metric.WithUnit("By"),
	))

	requesterResponseBytes = telemetry.Must(Meter.Float64Histogram(
		"ncl.requester.response.bytes",
		metric.WithDescription("Size of request responses"),
		metric.WithUnit("By"),
	))

	// Async publisher metrics
	asyncPublishEnqueueDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.async.enqueue.duration",
		metric.WithDescription("Time taken for async publish preparation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	asyncPublishEnqueuePartDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.async.enqueue.part.duration",
		metric.WithDescription("Time taken for sub-operations within async publish"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	asyncPublishEnqueueCount = telemetry.Must(Meter.Int64Counter(
		"ncl.publisher.async.enqueue.count",
		metric.WithDescription("Number of async publish operations"),
		metric.WithUnit("1"),
	))

	asyncPublishProcessDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.async.process.duration",
		metric.WithDescription("Time taken for processing queued messages"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	asyncPublishProcessPartDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.async.process.part.duration",
		metric.WithDescription("Time taken for sub-operations within message processing"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	asyncPublishProcessCount = telemetry.Must(Meter.Int64Counter(
		"ncl.publisher.async.process.count",
		metric.WithDescription("Number of messages processed from queue"),
		metric.WithUnit("1"),
	))

	asyncPublishCallbackDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.async.callback.duration",
		metric.WithDescription("Time taken for handling responses"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	asyncPublishCallbackPartDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.async.callback.part.duration",
		metric.WithDescription("Time taken for sub-operations within response handling"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	asyncPublishCallbackCount = telemetry.Must(Meter.Int64Counter(
		"ncl.publisher.async.callback.count",
		metric.WithDescription("Number of responses handled"),
		metric.WithUnit("1"),
	))

	asyncPublishTimeoutLoopDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.publisher.async.timeout.loop.duration",
		metric.WithDescription("Time taken for each timeout check loop"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	asyncPublishTimeoutCount = telemetry.Must(Meter.Int64Counter(
		"ncl.publisher.async.timeout.count",
		metric.WithDescription("Number of messages that timed out"),
		metric.WithUnit("1"),
	))

	asyncPublishInflightGauge = telemetry.Must(Meter.Int64ObservableGauge(
		"ncl.publisher.async.inflight.count",
		metric.WithDescription("Current number of inflight messages"),
		metric.WithUnit("1"),
	))

	asyncPublishQueueGauge = telemetry.Must(Meter.Int64ObservableGauge(
		"ncl.publisher.async.queue.count",
		metric.WithDescription("Current depth of message queue"),
		metric.WithUnit("1"),
	))

	// Subscriber metrics
	messageProcessDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.subscriber.process.duration",
		metric.WithDescription("Time taken for message processing"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	messageProcessPartDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.subscriber.process.part.duration",
		metric.WithDescription("Time taken for sub-operations within a message process"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	messageLatency = telemetry.Must(Meter.Float64Histogram(
		"ncl.subscriber.message.latency",
		metric.WithDescription("Time between message creation and reception"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	messageReceived = telemetry.Must(Meter.Int64Counter(
		"ncl.subscriber.message.count",
		metric.WithDescription("Number of messages received"),
		metric.WithUnit("1"),
	))

	messageBytes = telemetry.Must(Meter.Float64Histogram(
		"ncl.subscriber.message.bytes",
		metric.WithDescription("Size of received messages"),
		metric.WithUnit("By"),
	))

	// Responder metrics
	responderDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.responder.duration",
		metric.WithDescription("Time taken for request handling"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	responderPartDuration = telemetry.Must(Meter.Float64Histogram(
		"ncl.responder.part.duration",
		metric.WithDescription("Time taken for sub-operations within request handling"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(telemetry.DurationMsBuckets...),
	))

	responderCount = telemetry.Must(Meter.Int64Counter(
		"ncl.responder.request.count",
		metric.WithDescription("Number of requests handled"),
		metric.WithUnit("1"),
	))

	responderRequestBytes = telemetry.Must(Meter.Float64Histogram(
		"ncl.responder.request.bytes",
		metric.WithDescription("Size of received requests"),
		metric.WithUnit("By"),
	))

	responderResponseBytes = telemetry.Must(Meter.Float64Histogram(
		"ncl.responder.response.bytes",
		metric.WithDescription("Size of sent responses"),
		metric.WithUnit("By"),
	))
)

// Common attribute keys
const (
	AttrMessageType = "message_type"
	AttrInstance    = "instance"
	AttrSource      = "source"
	AttrOutcome     = "outcome"

	// Outcomes
	OutcomeSuccess    = "success"
	OutcomeFailure    = "failure"
	OutcomeFiltered   = "filtered"
	OutcomeAckFailure = "ack_failure"
	OutcomeCancelled  = "cancelled"
)
