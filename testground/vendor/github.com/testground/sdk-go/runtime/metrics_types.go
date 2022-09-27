package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
)

var pools [7]sync.Pool

func init() {
	for i := range pools {
		pools[i].New = func() interface{} {
			return &Metric{
				Type:     MetricType(i),
				Measures: make(map[string]interface{}, 1),
			}
		}
	}
}

type MetricType int

const (
	MetricPoint MetricType = iota
	MetricCounter
	MetricEWMA
	MetricGauge
	MetricHistogram
	MetricMeter
	MetricTimer
)

var typeMappings = [...]string{"point", "counter", "ewma", "gauge", "histogram", "meter", "timer"}

func (mt MetricType) String() string {
	return typeMappings[mt]
}

func (mt MetricType) MarshalJSON() ([]byte, error) {
	return json.Marshal(mt.String())
}

// UnmarshalJSON is only used for testing; it's inefficient but not relevant.
func (mt *MetricType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	for i, m := range typeMappings {
		if m == s {
			*mt = MetricType(i)
			return nil
		}
	}
	return fmt.Errorf("invalid metric type")
}

type Metric struct {
	Timestamp int64                  `json:"ts"`
	Type      MetricType             `json:"type"`
	Name      string                 `json:"name"`
	Measures  map[string]interface{} `json:"measures"`
}

func (m *Metric) Release() {
	pools[m.Type].Put(m)
}

func NewMetric(name string, i interface{}) *Metric {
	var (
		m  *Metric
		t  MetricType
		ts = time.Now().UnixNano()
	)

	switch v := i.(type) {
	case Point:
		t = MetricPoint
		m = pools[t].Get().(*Metric)
		m.Measures["value"] = float64(v)

	case Counter:
		t = MetricCounter
		m = pools[t].Get().(*Metric)
		s := v.Snapshot()
		m.Measures["count"] = s.Count()

	case EWMA:
		t = MetricEWMA
		m = pools[t].Get().(*Metric)
		s := v.Snapshot()
		m.Measures["rate"] = s.Rate()

	case Gauge: // float64 gauge, aliased in our SDK
		t = MetricGauge
		m = pools[t].Get().(*Metric)
		s := v.Snapshot()
		m.Measures["value"] = s.Value()

	case metrics.Gauge: // int64 gauge, used by go runtime metrics
		t = MetricGauge
		m = pools[t].Get().(*Metric)
		s := v.Snapshot()
		m.Measures["value"] = float64(s.Value())

	case Histogram:
		t = MetricHistogram
		m = pools[t].Get().(*Metric)
		s := v.Snapshot()
		p := s.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
		m.Measures["count"] = float64(s.Count())
		m.Measures["max"] = float64(s.Max())
		m.Measures["mean"] = s.Mean()
		m.Measures["min"] = float64(s.Min())
		m.Measures["stddev"] = s.StdDev()
		m.Measures["variance"] = s.Variance()
		m.Measures["p50"] = p[0]
		m.Measures["p75"] = p[1]
		m.Measures["p95"] = p[2]
		m.Measures["p99"] = p[3]
		m.Measures["p999"] = p[4]
		m.Measures["p9999"] = p[5]

	case Meter:
		t = MetricMeter
		m = pools[t].Get().(*Metric)
		s := v.Snapshot()
		m.Measures["count"] = float64(s.Count())
		m.Measures["m1"] = s.Rate1()
		m.Measures["m5"] = s.Rate5()
		m.Measures["m15"] = s.Rate15()
		m.Measures["mean"] = s.RateMean()

	case Timer:
		t = MetricTimer
		m = pools[t].Get().(*Metric)
		s := v.Snapshot()
		p := s.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
		m.Measures["count"] = float64(s.Count())
		m.Measures["max"] = float64(s.Max())
		m.Measures["mean"] = s.Mean()
		m.Measures["min"] = float64(s.Min())
		m.Measures["stddev"] = s.StdDev()
		m.Measures["variance"] = s.Variance()
		m.Measures["p50"] = p[0]
		m.Measures["p75"] = p[1]
		m.Measures["p95"] = p[2]
		m.Measures["p99"] = p[3]
		m.Measures["p999"] = p[4]
		m.Measures["p9999"] = p[5]
		m.Measures["m1"] = s.Rate1()
		m.Measures["m5"] = s.Rate5()
		m.Measures["m15"] = s.Rate15()
		m.Measures["meanrate"] = s.RateMean()

	default:
		panic(fmt.Sprintf("unexpected metric type: %v", reflect.TypeOf(v)))

	}

	m.Timestamp = ts
	m.Type = t
	m.Name = name
	return m
}
