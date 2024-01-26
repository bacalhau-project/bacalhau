package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Gauge is a synchronous Instrument which supports increments and decrements.
// Note: if the value is monotonically increasing, use Counter instead.
// Example uses for Gauge:
// - the number of active requests
// - the number of items in a queue
type Gauge struct {
	gauge metric.Int64UpDownCounter
}

func NewGauge(meter metric.Meter, name string, description string) (*Gauge, error) {
	gauge, err := meter.Int64UpDownCounter(name, metric.WithDescription(description))
	if err != nil {
		return nil, err
	}

	return &Gauge{
		gauge: gauge,
	}, nil
}

func (g *Gauge) Inc(ctx context.Context, attrs ...attribute.KeyValue) {
	g.gauge.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (g *Gauge) Dec(ctx context.Context, attrs ...attribute.KeyValue) {
	g.gauge.Add(ctx, -1, metric.WithAttributes(attrs...))
}
