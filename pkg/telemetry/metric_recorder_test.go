//go:build unit || !integration

package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

// MockFloat64Histogram implements metric.Float64Histogram for testing
type MockFloat64Histogram struct {
	embedded.Float64Histogram
	mock.Mock
}

func (m *MockFloat64Histogram) Record(ctx context.Context, value float64, opts ...metric.RecordOption) {
	m.Called(ctx, value, opts)
}

// MockInt64Counter implements metric.Int64Counter for testing
type MockInt64Counter struct {
	embedded.Int64Counter
	mock.Mock
}

func (m *MockInt64Counter) Add(ctx context.Context, value int64, opts ...metric.AddOption) {
	m.Called(ctx, value, opts)
}

// MockFloat64UpDownCounter implements metric.Float64UpDownCounter for testing
type MockFloat64UpDownCounter struct {
	embedded.Float64UpDownCounter
	mock.Mock
}

func (m *MockFloat64UpDownCounter) Add(ctx context.Context, value float64, opts ...metric.AddOption) {
	m.Called(ctx, value, opts)
}

type MetricRecorderTestSuite struct {
	suite.Suite
	ctx       context.Context
	mockClock *clock.Mock
	recorder  *MetricRecorder
	baseTime  time.Time
}

func (s *MetricRecorderTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.baseTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s.mockClock = clock.NewMock()
	s.mockClock.Set(s.baseTime)

	s.recorder = NewMetricRecorder(attribute.String("test", "true"))
	s.recorder.withClock(s.mockClock)
}

func (s *MetricRecorderTestSuite) TestNewMetricRecorder() {
	recorder := NewMetricRecorder(attribute.String("test", "value"))
	recorder.withClock(s.mockClock)

	s.NotNil(recorder)
	s.Len(recorder.attrs, 1)
	s.Equal("test", string(recorder.attrs[0].Key))
	s.Equal("value", recorder.attrs[0].Value.AsString())
	s.Equal(s.baseTime, recorder.start)
	s.Equal(s.baseTime, recorder.lastOperation)
	s.Empty(recorder.latencies)
	s.Empty(recorder.counts)
}

func (s *MetricRecorderTestSuite) TestWithAttributes() {
	s.recorder.WithAttributes(attribute.String("new", "attr"))

	s.Len(s.recorder.attrs, 2)
	s.Equal("new", string(s.recorder.attrs[1].Key))
	s.Equal("attr", s.recorder.attrs[1].Value.AsString())
}

func (s *MetricRecorderTestSuite) TestAddAttributes() {
	s.recorder.AddAttributes(attribute.String("new", "attr"))

	s.Len(s.recorder.attrs, 2)
	s.Equal("new", string(s.recorder.attrs[1].Key))
	s.Equal("attr", s.recorder.attrs[1].Value.AsString())
}

func (s *MetricRecorderTestSuite) TestResetLastOperation() {
	newTime := s.baseTime.Add(time.Second)
	s.mockClock.Set(newTime)
	s.recorder.ResetLastOperation()

	s.Equal(newTime, s.recorder.lastOperation)
}

func (s *MetricRecorderTestSuite) TestErrorString() {
	s.recorder.ErrorString("test_error")

	s.Len(s.recorder.attrs, 2)
	s.Equal(semconv.ErrorTypeKey, s.recorder.attrs[1].Key)
	s.Equal("test_error", s.recorder.attrs[1].Value.AsString())
}

func (s *MetricRecorderTestSuite) TestError() {
	err := bacerrors.New("test error").WithCode(bacerrors.IOError)
	s.recorder.Error(err)

	s.Len(s.recorder.attrs, 2)
	s.Equal(semconv.ErrorTypeKey, s.recorder.attrs[1].Key)
	s.Equal(string(bacerrors.IOError), s.recorder.attrs[1].Value.AsString())
}

func (s *MetricRecorderTestSuite) TestUnknownError() {
	err := errors.New("test error")
	s.recorder.Error(err)

	s.Len(s.recorder.attrs, 2)
	s.Equal(semconv.ErrorTypeKey, s.recorder.attrs[1].Key)
	s.Equal("unknown_error", s.recorder.attrs[1].Value.AsString())
}

func (s *MetricRecorderTestSuite) TestCount() {
	counter := new(MockInt64Counter)
	counter2 := new(MockInt64Counter)

	s.recorder.Count(s.ctx, counter)
	s.recorder.Count(s.ctx, counter)
	s.recorder.Count(s.ctx, counter2)

	s.Equal(int64(2), s.recorder.counts[counter])
	s.Equal(int64(1), s.recorder.counts[counter2])
}

func (s *MetricRecorderTestSuite) TestCountN() {
	counter := new(MockInt64Counter)
	counter2 := new(MockInt64Counter)

	s.recorder.CountN(s.ctx, counter, 5)
	s.recorder.CountN(s.ctx, counter, 3)
	s.recorder.CountN(s.ctx, counter2, 20)

	s.Equal(int64(8), s.recorder.counts[counter])
	s.Equal(int64(20), s.recorder.counts[counter2])
}

func (s *MetricRecorderTestSuite) TestGauge() {
	gauge := new(MockFloat64UpDownCounter)
	gauge.On("Add", s.ctx, float64(42), mock.Anything).Return()
	gauge.On("Add", s.ctx, float64(20), mock.Anything).Return()

	s.recorder.Gauge(s.ctx, gauge, 42)
	s.recorder.Gauge(s.ctx, gauge, 20)

	gauge.AssertExpectations(s.T())
}

func (s *MetricRecorderTestSuite) TestHistogram() {
	hist := new(MockFloat64Histogram)

	// Record multiple values
	s.recorder.Histogram(s.ctx, hist, 10.5)
	s.recorder.Histogram(s.ctx, hist, 20.3)
	s.recorder.Histogram(s.ctx, hist, 5.7)

	// Verify values are aggregated but not yet recorded
	s.Len(s.recorder.histograms, 1)
	s.Equal(36.5, s.recorder.histograms[hist]) // 10.5 + 20.3 + 5.7 = 36.5
	hist.AssertNotCalled(s.T(), "Record")

	// Different histogram should get separate aggregation
	hist2 := new(MockFloat64Histogram)
	s.recorder.Histogram(s.ctx, hist2, 15.0)
	s.recorder.Histogram(s.ctx, hist2, 25.0)

	s.Len(s.recorder.histograms, 2)
	s.Equal(36.5, s.recorder.histograms[hist])
	s.Equal(40.0, s.recorder.histograms[hist2])
	hist2.AssertNotCalled(s.T(), "Record")
}

func (s *MetricRecorderTestSuite) TestCountAndHistogram() {
	counter := new(MockInt64Counter)
	hist := new(MockFloat64Histogram)

	// Record multiple values
	s.recorder.CountAndHistogram(s.ctx, counter, hist, 10.5)
	s.recorder.CountAndHistogram(s.ctx, counter, hist, 20.3)

	// Verify counter is incremented
	s.Equal(int64(30), s.recorder.counts[counter]) // 10 + 20 = 30 (truncated to int64)

	// Verify histogram values are aggregated
	s.Equal(30.8, s.recorder.histograms[hist]) // 10.5 + 20.3 = 30.8

	// Verify nothing recorded yet
	counter.AssertNotCalled(s.T(), "Add")
	hist.AssertNotCalled(s.T(), "Record")
}

func (s *MetricRecorderTestSuite) TestDuration() {
	duration := time.Second
	hist := new(MockFloat64Histogram)

	s.recorder.Duration(s.ctx, hist, duration)

	// Should be aggregated, not recorded immediately
	s.Len(s.recorder.histograms, 1)
	s.Equal(1.0, s.recorder.histograms[hist])
	hist.AssertNotCalled(s.T(), "Record")

	// Add another duration
	s.recorder.Duration(s.ctx, hist, 2*time.Second)
	s.Equal(3.0, s.recorder.histograms[hist]) // 1.0 + 2.0 = 3.0
	hist.AssertNotCalled(s.T(), "Record")
}

func (s *MetricRecorderTestSuite) TestLatency() {
	hist1 := new(MockFloat64Histogram)
	hist2 := new(MockFloat64Histogram)

	// First measurement for hist1/"op1"
	s.mockClock.Add(100 * time.Millisecond)
	s.recorder.Latency(s.ctx, hist1, "op1")
	key1 := histogramKey{histogram: hist1, operation: "op1"}
	s.Len(s.recorder.latencies, 1)
	s.Equal(100*time.Millisecond, s.recorder.latencies[key1])

	// Second measurement for hist1/"op1" should aggregate
	s.mockClock.Add(150 * time.Millisecond)
	s.recorder.Latency(s.ctx, hist1, "op1")
	s.Len(s.recorder.latencies, 1)                            // Still just one entry
	s.Equal(250*time.Millisecond, s.recorder.latencies[key1]) // Aggregated duration

	// Different operation name gets new entry
	s.mockClock.Add(200 * time.Millisecond)
	s.recorder.Latency(s.ctx, hist1, "op2")
	key2 := histogramKey{histogram: hist1, operation: "op2"}
	s.Len(s.recorder.latencies, 2)                            // Now two entries
	s.Equal(250*time.Millisecond, s.recorder.latencies[key1]) // First op unchanged
	s.Equal(200*time.Millisecond, s.recorder.latencies[key2]) // New operation duration

	// Same operation name but different histogram gets separate entry
	s.mockClock.Add(300 * time.Millisecond)
	s.recorder.Latency(s.ctx, hist2, "op1")
	key3 := histogramKey{histogram: hist2, operation: "op1"}
	s.Len(s.recorder.latencies, 3)                            // Now three entries
	s.Equal(250*time.Millisecond, s.recorder.latencies[key1]) // First hist/op unchanged
	s.Equal(200*time.Millisecond, s.recorder.latencies[key2]) // First hist/op2 unchanged
	s.Equal(300*time.Millisecond, s.recorder.latencies[key3]) // New hist/op duration
}

func (s *MetricRecorderTestSuite) TestDone() {
	counter := new(MockInt64Counter)
	opHist := new(MockFloat64Histogram)
	valueHist := new(MockFloat64Histogram)
	durationHist := new(MockFloat64Histogram)

	// Record metrics
	s.mockClock.Add(100 * time.Millisecond)
	s.recorder.Count(s.ctx, counter)
	s.recorder.Latency(s.ctx, opHist, "op1")
	s.recorder.Histogram(s.ctx, valueHist, 42.5)
	s.recorder.Histogram(s.ctx, valueHist, 17.5)

	// Verify nothing published yet
	opHist.AssertNotCalled(s.T(), "Record")
	valueHist.AssertNotCalled(s.T(), "Record")
	counter.AssertNotCalled(s.T(), "Add")

	// Add some more timing
	s.mockClock.Add(150 * time.Millisecond)

	// Setup expectations with precise timing and attribute validation
	durationHist.On("Record", s.ctx, float64(0.25), mock.MatchedBy(func(opts []metric.RecordOption) bool {
		cfg := metric.NewRecordConfig(opts)
		set := cfg.Attributes()
		return set.HasValue("test") && set.HasValue("final")
	})).Return().Once() // Total duration: 250ms

	opHist.On("Record", s.ctx, float64(0.1), mock.MatchedBy(func(opts []metric.RecordOption) bool {
		cfg := metric.NewRecordConfig(opts)
		set := cfg.Attributes()
		return set.HasValue("test") &&
			set.HasValue("final") &&
			set.HasValue("sub_operation")
	})).Return().Once() // op1 latency: 100ms

	valueHist.On("Record", s.ctx, float64(60.0), mock.MatchedBy(func(opts []metric.RecordOption) bool {
		cfg := metric.NewRecordConfig(opts)
		set := cfg.Attributes()
		return set.HasValue("test") && set.HasValue("final")
	})).Return().Once() // histogram sum: 42.5 + 17.5 = 60.0

	counter.On("Add", s.ctx, int64(1), mock.MatchedBy(func(opts []metric.AddOption) bool {
		cfg := metric.NewAddConfig(opts)
		set := cfg.Attributes()
		return set.HasValue("test") && set.HasValue("final")
	})).Return().Once()

	// Call Done with additional attributes
	s.recorder.Done(s.ctx, durationHist, attribute.String("final", "attr"))

	// Verify expectations
	opHist.AssertExpectations(s.T())
	valueHist.AssertExpectations(s.T())
	durationHist.AssertExpectations(s.T())
	counter.AssertExpectations(s.T())

	// Verify state was reset
	s.Empty(s.recorder.latencies)
	s.Empty(s.recorder.counts)
	s.Empty(s.recorder.histograms)
	s.Equal(s.mockClock.Now(), s.recorder.start)
	s.Equal(s.mockClock.Now(), s.recorder.lastOperation)
}

func (s *MetricRecorderTestSuite) TestDoneWithMultipleOperations() {
	counter := new(MockInt64Counter)
	hist1 := new(MockFloat64Histogram)
	hist2 := new(MockFloat64Histogram)
	valueHist := new(MockFloat64Histogram)
	durationHist := new(MockFloat64Histogram)

	// First round of operations
	s.mockClock.Add(100 * time.Millisecond)
	s.recorder.Count(s.ctx, counter)
	s.recorder.Latency(s.ctx, hist1, "op1") // 100ms in hist1/op1
	s.recorder.Histogram(s.ctx, valueHist, 42.5)
	s.mockClock.Add(150 * time.Millisecond)
	s.recorder.Latency(s.ctx, hist1, "op1") // +150ms in hist1/op1
	s.recorder.Histogram(s.ctx, valueHist, 17.5)
	s.mockClock.Add(200 * time.Millisecond)
	s.recorder.Latency(s.ctx, hist2, "op1") // 200ms in hist2/op1

	// Setup expectations for first Done
	hist1.On("Record", s.ctx, float64(0.25), mock.Anything).Return().Once()        // hist1/op1 total: 250ms
	hist2.On("Record", s.ctx, float64(0.2), mock.Anything).Return().Once()         // hist2/op1: 200ms
	valueHist.On("Record", s.ctx, float64(60.0), mock.Anything).Return().Once()    // valueHist: 42.5 + 17.5 = 60.0
	durationHist.On("Record", s.ctx, float64(0.45), mock.Anything).Return().Once() // Total: 450ms
	counter.On("Add", s.ctx, int64(1), mock.Anything).Return().Once()

	// First Done call
	s.recorder.Done(s.ctx, durationHist)

	// Verify state was reset
	s.Empty(s.recorder.latencies)
	s.Empty(s.recorder.counts)
	s.Empty(s.recorder.histograms)
	s.Equal(s.mockClock.Now(), s.recorder.start)
	s.Equal(s.mockClock.Now(), s.recorder.lastOperation)

	// Second round of operations
	s.mockClock.Add(100 * time.Millisecond)
	s.recorder.Count(s.ctx, counter)
	s.recorder.Latency(s.ctx, hist1, "op2") // 100ms in hist1/op2
	s.recorder.Histogram(s.ctx, valueHist, 30.0)

	// Setup expectations for second Done
	hist1.On("Record", s.ctx, float64(0.1), mock.Anything).Return().Once()        // hist1/op2: 100ms
	valueHist.On("Record", s.ctx, float64(30.0), mock.Anything).Return().Once()   // valueHist: 30.0
	durationHist.On("Record", s.ctx, float64(0.1), mock.Anything).Return().Once() // Total: 100ms
	counter.On("Add", s.ctx, int64(1), mock.Anything).Return().Once()

	// Second Done call
	s.recorder.Done(s.ctx, durationHist)

	// Verify all expectations
	hist1.AssertExpectations(s.T())
	hist2.AssertExpectations(s.T())
	valueHist.AssertExpectations(s.T())
	durationHist.AssertExpectations(s.T())
	counter.AssertExpectations(s.T())
}

func TestMetricRecorderSuite(t *testing.T) {
	suite.Run(t, new(MetricRecorderTestSuite))
}
