package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Timer measures the duration of an event.
type Timer struct {
	startTime        time.Time
	durationRecorder metric.Int64Histogram
}

func NewTimer(durationRecorder metric.Int64Histogram) *Timer {
	return &Timer{
		durationRecorder: durationRecorder,
	}
}

// Start begins the timer by recording the current time.
func (t *Timer) Start() {
	t.startTime = time.Now()
}

// Stop ends the timer and records the duration since Start was called.
// `attrs` are optional attributes that can be added to the duration metric for additional context.
func (t *Timer) Stop(ctx context.Context, attrs ...attribute.KeyValue) {
	if t.startTime.IsZero() {
		// Handle the case where Stop is called without Start being called.
		return
	}

	// Calculate the duration and record it using the OpenTelemetry histogram.
	duration := time.Since(t.startTime).Milliseconds()
	t.durationRecorder.Record(ctx, duration, metric.WithAttributes(attrs...))
	t.startTime = time.Time{} // Reset the start time for future use.
}
