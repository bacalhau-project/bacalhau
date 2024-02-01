package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Timer is a function to record a duration. Calling it starts the timer,
// calling the returned function will record the duration.
func Timer(
	ctx context.Context,
	durationRecorder metric.Int64Histogram,
	attrs ...attribute.KeyValue,
) func() time.Duration {
	start := time.Now()
	return func() time.Duration {
		dur := time.Since(start)
		durationRecorder.Record(ctx, dur.Milliseconds(), metric.WithAttributes(attrs...))
		return dur
	}
}
