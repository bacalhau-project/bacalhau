package runtime

import (
	"math"
	"sort"
	"sync"

	"github.com/rcrowley/go-metrics"
)

// Initial slice capacity for the values stored in a ResettingHistogram
const InitialResettingHistogramSliceCap = 10

// newResettingHistogram constructs a new StandardResettingHistogram
func newResettingHistogram() Histogram {
	if metrics.UseNilMetrics {
		return nilResettingHistogram{}
	}
	return &standardResettingHistogram{
		values: make([]int64, 0, InitialResettingHistogramSliceCap),
	}
}

// nilResettingHistogram is a no-op ResettingHistogram.
type nilResettingHistogram struct {
}

// Values is a no-op.
func (nilResettingHistogram) Values() []int64 { return nil }

// Snapshot is a no-op.
func (nilResettingHistogram) Snapshot() Histogram {
	return &resettingHistogramSnapshot{
		values: []int64{},
	}
}

// Update is a no-op.
func (nilResettingHistogram) Update(int64) {}

// Clear is a no-op.
func (nilResettingHistogram) Clear() {}

func (nilResettingHistogram) Count() int64 {
	return 0
}

func (nilResettingHistogram) Variance() float64 {
	return 0.0
}

func (nilResettingHistogram) Min() int64 {
	return 0
}

func (nilResettingHistogram) Max() int64 {
	return 0
}

func (nilResettingHistogram) Sum() int64 {
	return 0
}

func (nilResettingHistogram) StdDev() float64 {
	return 0.0
}

func (nilResettingHistogram) Sample() Sample {
	return metrics.NilSample{}
}

func (nilResettingHistogram) Percentiles([]float64) []float64 {
	return nil
}

func (nilResettingHistogram) Percentile(float64) float64 {
	return 0.0
}

func (nilResettingHistogram) Mean() float64 {
	return 0.0
}

// standardResettingHistogram is used for storing aggregated values for timers, which are reset on every flush interval.
type standardResettingHistogram struct {
	values []int64
	mutex  sync.Mutex
}

func (t *standardResettingHistogram) Count() int64 {
	panic("Count called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Max() int64 {
	panic("Max called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Min() int64 {
	panic("Min called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) StdDev() float64 {
	panic("StdDev called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Variance() float64 {
	panic("Variance called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Sum() int64 {
	panic("Sum called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Sample() Sample {
	panic("Sample called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Percentiles([]float64) []float64 {
	panic("Percentiles called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Percentile(float64) float64 {
	panic("Percentile called on a resetting histogram; capture a snapshot first")
}

func (t *standardResettingHistogram) Mean() float64 {
	panic("Mean called on a resetting histogram; capture a snapshot first")
}

// Values returns a slice with all measurements.
func (t *standardResettingHistogram) Values() []int64 {
	return t.values
}

// Snapshot resets the timer and returns a read-only copy of its sorted contents.
func (t *standardResettingHistogram) Snapshot() Histogram {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	currentValues := t.values
	t.values = make([]int64, 0, InitialResettingHistogramSliceCap)

	sort.Slice(currentValues, func(i, j int) bool { return currentValues[i] < currentValues[j] })

	return &resettingHistogramSnapshot{
		values: currentValues,
	}
}

func (t *standardResettingHistogram) Clear() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.values = t.values[:0]
}

// Record the duration of an event.
func (t *standardResettingHistogram) Update(d int64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.values = append(t.values, d)
}

// resettingHistogramSnapshot is a point-in-time copy of another resettingHistogram.
type resettingHistogramSnapshot struct {
	sync.Mutex

	values     []int64
	mean       float64
	calculated bool
}

// resettingHistogramSnapshot returns the snapshot.
func (t *resettingHistogramSnapshot) Snapshot() Histogram { return t }

func (*resettingHistogramSnapshot) Update(int64) {
	panic("Update called on a resetting histogram snapshot")
}

func (t *resettingHistogramSnapshot) Clear() {
	panic("Clear called on a resetting histogram snapshot")
}

func (t *resettingHistogramSnapshot) Sample() Sample {
	panic("Sample called on a resetting histogram snapshot")
}

func (t *resettingHistogramSnapshot) Count() int64 {
	t.Lock()
	defer t.Unlock()

	return int64(len(t.values))
}

// Values returns all values from snapshot.
func (t *resettingHistogramSnapshot) Values() []int64 {
	t.Lock()
	defer t.Unlock()

	return t.values
}

func (t *resettingHistogramSnapshot) Min() int64 {
	t.Lock()
	defer t.Unlock()

	if len(t.values) > 0 {
		return t.values[0]
	}
	return 0
}

func (t *resettingHistogramSnapshot) Variance() float64 {
	t.Lock()
	defer t.Unlock()

	if len(t.values) == 0 {
		return 0.0
	}

	m := t._mean()
	var sum float64
	for _, v := range t.values {
		d := float64(v) - m
		sum += d * d
	}
	return sum / float64(len(t.values))
}

func (t *resettingHistogramSnapshot) Max() int64 {
	t.Lock()
	defer t.Unlock()

	if len(t.values) > 0 {
		return t.values[len(t.values)-1]
	}
	return 0
}

func (t *resettingHistogramSnapshot) StdDev() float64 {
	return math.Sqrt(t.Variance())
}

func (t *resettingHistogramSnapshot) Sum() int64 {
	t.Lock()
	defer t.Unlock()

	var sum int64
	for _, v := range t.values {
		sum += v
	}
	return sum
}

// Percentile returns the boundaries for the input percentiles.
func (t *resettingHistogramSnapshot) Percentile(percentile float64) float64 {
	t.Lock()
	defer t.Unlock()

	tb := t.calc([]float64{percentile})

	return tb[0]
}

// Percentiles returns the boundaries for the input percentiles.
func (t *resettingHistogramSnapshot) Percentiles(percentiles []float64) []float64 {
	t.Lock()
	defer t.Unlock()

	tb := t.calc(percentiles)

	return tb
}

// Mean returns the mean of the snapshotted values
func (t *resettingHistogramSnapshot) Mean() float64 {
	t.Lock()
	defer t.Unlock()

	return t._mean()
}

func (t *resettingHistogramSnapshot) _mean() float64 {
	if !t.calculated {
		_ = t.calc([]float64{})
	}

	return t.mean
}

func (t *resettingHistogramSnapshot) calc(percentiles []float64) (thresholdBoundaries []float64) {
	count := len(t.values)
	if count == 0 {
		thresholdBoundaries = make([]float64, len(percentiles))
		t.mean = 0
		t.calculated = true
		return
	}

	min := t.values[0]
	max := t.values[count-1]

	cumulativeValues := make([]int64, count)
	cumulativeValues[0] = min
	for i := 1; i < count; i++ {
		cumulativeValues[i] = t.values[i] + cumulativeValues[i-1]
	}

	thresholdBoundaries = make([]float64, len(percentiles))

	thresholdBoundary := max

	for i, pct := range percentiles {
		if count > 1 {
			var abs float64
			if pct >= 0 {
				abs = pct
			} else {
				abs = 100 + pct
			}
			// poor man's math.Round(x):
			// math.Floor(x + 0.5)
			indexOfPerc := int(math.Floor(((abs / 100.0) * float64(count)) + 0.5))
			if pct >= 0 && indexOfPerc > 0 {
				indexOfPerc -= 1 // index offset=0
			}
			thresholdBoundary = t.values[indexOfPerc]
		}

		thresholdBoundaries[i] = float64(thresholdBoundary)
	}

	sum := cumulativeValues[count-1]
	t.mean = float64(sum) / float64(count)
	t.calculated = true
	return
}
