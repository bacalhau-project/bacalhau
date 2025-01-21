package telemetry

import (
	"context"
	"time"

	"github.com/benbjohnson/clock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

var (
	SubOperationKey = attribute.Key("sub_operation")
)

type histogramKey struct {
	histogram metric.Float64Histogram
	operation string
}

// MetricRecorder is a helper for recording metrics.
// It provides methods to record latency, counters, and gauges with consistent attribute handling.
// The recorder aggregates metrics internally and only publishes them when Done() is called.
//
// IMPORTANT: MetricRecorder is not thread-safe. It should only be used by a single goroutine.
// If you need to record metrics from multiple goroutines, create separate recorders for each.
//
// Example usage:
//
//	recorder := NewMetricRecorder(attribute.String("operation", "process"))
//	defer recorder.Done(ctx, durationHist) // Records total duration and all aggregated metrics
//
//	// In a loop:
//	msg := queue.Receive()
//	recorder.Latency(ctx, dequeueHist, "dequeue") // Aggregates dequeue time
//
//	if err := process(msg); err != nil {
//	    recorder.Error("processing_failed")
//	    return err
//	}
//	recorder.Latency(ctx, processHist, "process") // Aggregates process time
//	recorder.Count(ctx, successCounter) // Aggregates success count
type MetricRecorder struct {
	start         time.Time
	lastOperation time.Time
	attrs         []attribute.KeyValue

	// Internal aggregation state - not thread safe
	latencies map[histogramKey]time.Duration // Aggregated latencies by histogram+operation
	counts    map[metric.Int64Counter]int64  // Aggregated counts by counter

	// Clock used for time operations - can be overridden for testing
	clock clock.Clock
}

// NewMetricRecorder creates a new recorder with base attributes that will be included
// in all metric recordings. The recorder will aggregate metrics until Done() is called.
func NewMetricRecorder(attrs ...attribute.KeyValue) *MetricRecorder {
	r := &MetricRecorder{
		attrs: attrs,
		clock: clock.New(),
	}
	r.reset()
	return r
}

// WithAttributes adds attributes for all future measurements.
func (r *MetricRecorder) WithAttributes(attrs ...attribute.KeyValue) *MetricRecorder {
	r.attrs = append(r.attrs, attrs...)
	return r
}

// withClock sets the clock used for time operations.
//
//nolint:unused
func (r *MetricRecorder) withClock(clock clock.Clock) *MetricRecorder {
	r.clock = clock
	r.reset()
	return r
}

// AddAttributes adds attributes for all future measurements.
func (r *MetricRecorder) AddAttributes(attrs ...attribute.KeyValue) {
	r.attrs = append(r.attrs, attrs...)
}

// ResetLastOperation resets the timestamp used for next Latency calculation
func (r *MetricRecorder) ResetLastOperation() {
	r.lastOperation = r.clock.Now()
}

// ErrorString records an error type using OpenTelemetry semantic conventions
func (r *MetricRecorder) ErrorString(typ string) {
	r.attrs = append(r.attrs, semconv.ErrorTypeKey.String(typ))
}

// Error records an error type using OpenTelemetry semantic conventions
func (r *MetricRecorder) Error(err error) {
	if err == nil {
		return
	}
	if bErr := bacerrors.FromError(err); bErr != nil {
		r.ErrorString(string(bErr.Code()))
		return
	}
	r.ErrorString(err.Error())
}

// Count aggregates counter increments by 1.
// The aggregated count will be published with base attributes when Done() is called.
func (r *MetricRecorder) Count(ctx context.Context, c metric.Int64Counter) {
	r.counts[c]++
}

// CountN aggregates counter increments by n.
// The aggregated count will be published with base attributes when Done() is called.
func (r *MetricRecorder) CountN(ctx context.Context, c metric.Int64Counter, n int64) {
	r.counts[c] += n
}

// Gauge sets gauge value. Unlike Count and Latency, gauge values are published immediately.
func (r *MetricRecorder) Gauge(ctx context.Context, g metric.Float64UpDownCounter, val float64) {
	g.Add(ctx, val, metric.WithAttributes(r.attrs...))
}

// Duration records a duration for a histogram. Unlike Latency, durations are published immediately.
func (r *MetricRecorder) Duration(ctx context.Context, h metric.Float64Histogram, duration time.Duration) {
	h.Record(ctx, duration.Seconds(), metric.WithAttributes(r.attrs...))
}

// Latency aggregates the time since the last operation or start for a given sub-operation.
// If this is the first operation, it records the latency since start.
// The aggregated latencies will be published when Done() is called.
func (r *MetricRecorder) Latency(ctx context.Context, h metric.Float64Histogram, subOperation string) {
	duration := r.clock.Since(r.lastOperation)
	r.lastOperation = r.clock.Now()
	key := histogramKey{histogram: h, operation: subOperation}
	r.latencies[key] += duration
}

// Done records the total duration since recorder creation and publishes all aggregated metrics.
// This should typically be deferred immediately after creating the recorder.
// Additional attributes can be provided and will be merged with base attributes.
func (r *MetricRecorder) Done(ctx context.Context, h metric.Float64Histogram, attrs ...attribute.KeyValue) {
	// Record total duration
	finalAttrs := append(r.attrs, attrs...)
	h.Record(ctx, r.clock.Since(r.start).Seconds(), metric.WithAttributes(finalAttrs...))

	// Publish all aggregated latencies to their respective histograms
	for key, duration := range r.latencies {
		opAttrs := append(finalAttrs, SubOperationKey.String(key.operation))
		key.histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(opAttrs...))
	}

	// Publish all aggregated counts
	for counter, value := range r.counts {
		counter.Add(ctx, value, metric.WithAttributes(finalAttrs...))
	}

	r.reset()
}

// reset resets the recorder for reuse.
func (r *MetricRecorder) reset() {
	r.start = r.clock.Now()
	r.lastOperation = r.start
	r.latencies = make(map[histogramKey]time.Duration)
	r.counts = make(map[metric.Int64Counter]int64)
}
