package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricRecorder is a helper for recording metrics.
// Specifically, it provides methods to record latency and faults.
//
// Example usage:
//
//	recorder := NewMetricRecorder()
//	defer recorder.RecordFault(ctx, faultCounter) // Records fault if success was not reported
//	defer recorder.RecordTotalLatency(ctx, totalLatency) // Records total latency since start
//
//	// Alternatively, you can use the combined method to record fault and total latency
//	// defer recorder.RecordFaultAndLatency(ctx, faultCounter, totalLatency)
//
//	msg := queue.Receive()
//	recorder.RecordLatency(ctx, dequeueLatency) // Records dequeue latency since start
//
//	process(msg)
//	recorder.RecordLatency(ctx, processLatency) // Records process latency since queue.Receive()
//
//	recorder.Success()
type MetricRecorder struct {
	start         time.Time
	lastOperation time.Time
	success       bool
}

func NewMetricRecorder() *MetricRecorder {
	now := time.Now()
	return &MetricRecorder{
		start:         now,
		lastOperation: now,
	}
}

// RecordLatency records the latency since the last operation.
// If this the first operation, it records the latency since the start.
func (t *MetricRecorder) RecordLatency(ctx context.Context, histogram metric.Int64Histogram, options ...metric.RecordOption) {
	latency := time.Since(t.lastOperation)
	t.lastOperation = time.Now()
	histogram.Record(ctx, latency.Milliseconds(), options...)
}

// RecordTotalLatency records the latency since the start.
func (t *MetricRecorder) RecordTotalLatency(ctx context.Context, histogram metric.Int64Histogram, options ...metric.RecordOption) {
	latency := time.Since(t.start)
	histogram.Record(ctx, latency.Milliseconds(), options...)
}

// RecordEvent records an event
func (t *MetricRecorder) RecordEvent(ctx context.Context, counter metric.Int64Counter, options ...metric.AddOption) {
	var fault int64 = 1
	if t.success {
		fault = 0
	}

	m := make(map[string]string)

	var keyvals []attribute.KeyValue
	for k, v := range m {
		keyvals = append(keyvals, attribute.String(k, v))
	}

	counter.Add(ctx, fault, metric.WithAttributes(keyvals...))
}

// RecordFault records a fault as 1 if success was not reported, 0 otherwise.
func (t *MetricRecorder) RecordFault(ctx context.Context, counter metric.Int64Counter, options ...metric.AddOption) {
	var fault int64 = 1
	if t.success {
		fault = 0
	}
	counter.Add(ctx, fault, options...)
}

// RecordFaultAndLatency records a fault and total latency.
func (t *MetricRecorder) RecordFaultAndLatency(
	ctx context.Context, counter metric.Int64Counter, histogram metric.Int64Histogram, options ...metric.MeasurementOption) {
	addOptions := make([]metric.AddOption, len(options))
	for i, option := range options {
		addOptions[i] = option
	}
	t.RecordFault(ctx, counter, addOptions...)

	recordOptions := make([]metric.RecordOption, len(options))
	for i, option := range options {
		recordOptions[i] = option
	}
	t.RecordTotalLatency(ctx, histogram, recordOptions...)
}

// Success records that the operation was successful.
// It doesn't record anything, but it is used to determine whether a fault should be recorded.
func (t *MetricRecorder) Success() {
	t.success = true
}
