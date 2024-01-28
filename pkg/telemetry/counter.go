package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Counter is a synchronous Instrument which supports non-negative increments
// Example uses for Counter:
// - count the number of bytes received
// - count the number of requests completed
// - count the number of accounts created
// - count the number of checkpoints run
// - count the number of HTTP 5xx errors
type Counter struct {
	counter metric.Int64Counter
}

func NewCounter(meter metric.Meter, name string, description string) (*Counter, error) {
	counter, err := meter.Int64Counter(name, metric.WithDescription(description))
	if err != nil {
		return nil, err
	}

	return &Counter{
		counter: counter,
	}, nil
}

func (c *Counter) Inc(ctx context.Context, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (c *Counter) Add(ctx context.Context, num int64, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, num, metric.WithAttributes(attrs...))
}
