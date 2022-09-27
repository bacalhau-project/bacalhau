package runtime

import (
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
)

// Type aliases to hide implementation details in the APIs.
type (
	Counter   = metrics.Counter
	EWMA      = metrics.EWMA
	Gauge     = metrics.GaugeFloat64
	Histogram = metrics.Histogram
	Meter     = metrics.Meter
	Sample    = metrics.Sample
	Timer     = metrics.Timer
	Point     float64
)

type MetricSinkFn func(m *Metric) error

type MetricsApi struct {
	// re is the RunEnv this MetricsApi object is attached to.
	re *RunEnv

	// reg is the go-metrics Registry this MetricsApi object creates metrics under.
	reg metrics.Registry

	// sinks to invoke when a new observation has been made.
	//  1) data points are sent immediately.
	//  2) aggregated metrics are sent periodically, based on freq.
	sinks []MetricSinkFn

	// freq is the frequency with which to materialize aggregated metrics.
	freq time.Duration

	// callbacks are callbacks functions to call on every tick.
	callbacks []func(registry metrics.Registry)

	wg           sync.WaitGroup
	freqChangeCh chan time.Duration
	doneCh       chan struct{}
}

type metricsApiOpts struct {
	freq        time.Duration
	preregister func(registry metrics.Registry)
	callbacks   []func(registry metrics.Registry)
	sinks       []MetricSinkFn
}

func newMetricsApi(re *RunEnv, opts metricsApiOpts) *MetricsApi {
	m := &MetricsApi{
		re:           re,
		reg:          metrics.NewRegistry(),
		sinks:        opts.sinks,
		freq:         opts.freq,
		callbacks:    opts.callbacks,
		freqChangeCh: make(chan time.Duration),
		doneCh:       make(chan struct{}),
	}

	if opts.preregister != nil {
		opts.preregister(m.reg)
	}

	m.wg.Add(1)
	go m.background()
	return m
}

func (m *MetricsApi) background() {
	var (
		tick *time.Ticker
		c    <-chan time.Time
	)

	defer m.wg.Done()

	// resetTicker resets the ticker to a new frequency.
	resetTicker := func(d time.Duration) {
		if tick != nil {
			tick.Stop()
			tick = nil
			c = nil
		}
		if d <= 0 {
			return
		}
		tick = time.NewTicker(d)
		c = tick.C
	}

	// Will stop and nullify the ticker.
	defer resetTicker(0)

	// Set the initial tick frequency.
	resetTicker(m.freq)

	for {
		select {
		case <-c:
			for _, a := range m.callbacks {
				a(m.reg)
			}
			m.reg.Each(m.broadcast)

		case f := <-m.freqChangeCh:
			m.freq = f
			resetTicker(f)

		case <-m.doneCh:
			return
		}
	}
}

// broadcast sends an observation to all emitters.
func (m *MetricsApi) broadcast(name string, obj interface{}) {
	metric := NewMetric(name, obj)
	defer metric.Release()

	for _, sink := range m.sinks {
		if err := sink(metric); err != nil {
			m.re.RecordMessage("failed to emit metric: %s", err)
		}
	}
}

func (m *MetricsApi) Close() error {
	close(m.doneCh)
	m.wg.Wait()

	return nil
}

func (m *MetricsApi) SetFrequency(freq time.Duration) {
	m.freqChangeCh <- freq
}

// RecordPoint records a float64 point under the provided metric name + tags.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) RecordPoint(name string, value float64) {
	m.broadcast(name, Point(value))
}

// Counter creates a measurement of counter type. The returned type is an
// alias of go-metrics' Counter type. Refer to godocs there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) Counter(name string) Counter {
	return m.reg.GetOrRegister(name, newResettingCounter()).(metrics.Counter)
}

// EWMA creates a measurement of exponential-weighted moving average type.
// The returned type is an alias of go-metrics' EWMA type. Refer to godocs
// there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) EWMA(name string, alpha float64) EWMA {
	return m.reg.GetOrRegister(name, metrics.NewEWMA(alpha)).(metrics.EWMA)
}

// Gauge creates a measurement of gauge type (float64).
// The returned type is an alias of go-metrics' GaugeFloat64 type. Refer to
// godocs there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) Gauge(name string) Gauge {
	return m.reg.GetOrRegister(name, metrics.NewGaugeFloat64()).(metrics.GaugeFloat64)
}

// GaugeF creates a measurement of functional gauge type (float64).
// The returned type is an alias of go-metrics' GaugeFloat64 type. Refer to
// godocs there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) GaugeF(name string, f func() float64) Gauge {
	return m.reg.GetOrRegister(name, metrics.NewFunctionalGaugeFloat64(f)).(metrics.GaugeFloat64)
}

// Histogram creates a measurement of histogram type.
// The returned type is an alias of go-metrics' Histogram type. Refer to
// godocs there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) Histogram(name string, s Sample) Histogram {
	return m.reg.GetOrRegister(name, metrics.NewHistogram(s)).(metrics.Histogram)
}

// Meter creates a measurement of meter type.
// The returned type is an alias of go-metrics' Meter type. Refer to
// godocs there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) Meter(name string) Meter {
	return m.reg.GetOrRegister(name, metrics.NewMeter()).(metrics.Meter)
}

// Timer creates a measurement of timer type.
// The returned type is an alias of go-metrics' Timer type. Refer to
// godocs there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) Timer(name string) Timer {
	return m.reg.GetOrRegister(name, metrics.NewTimer()).(metrics.Timer)
}

// ResettingHistogram creates a measurement of histogram type, which cyclically
// resets to zero when its values are harvested.
//
// The returned type is an alias of go-metrics' Histogram type. Refer to
// godocs there for details.
//
// The format of the metric name is a comma-separated list, where the first
// element is the metric name, and optionally, an unbounded list of
// key-value pairs. Example:
//
//   requests_received,tag1=value1,tag2=value2,tag3=value3
func (m *MetricsApi) ResettingHistogram(name string) Histogram {
	return m.reg.GetOrRegister(name, newResettingHistogram()).(Histogram)
}

func (m *MetricsApi) NewExpDecaySample(reservoirSize int, alpha float64) Sample {
	return metrics.NewExpDecaySample(reservoirSize, alpha)
}

func (m *MetricsApi) NewUniformSample(reservoirSize int) Sample {
	return metrics.NewUniformSample(reservoirSize)
}
