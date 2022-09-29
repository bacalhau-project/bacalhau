package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	_ "github.com/influxdata/influxdb1-client"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/rcrowley/go-metrics"
)

type Metrics struct {
	re          *RunEnv
	diagnostics *MetricsApi
	results     *MetricsApi
	influxdb    client.Client
	batcher     Batcher
	tags        map[string]string
}

func newMetrics(re *RunEnv) *Metrics {
	m := &Metrics{re: re}

	var dsinks = []MetricSinkFn{m.logSinkJSON("diagnostics.out")}

	if re.TestDisableMetrics {
		re.RecordMessage("InfluxDB batching disabled by test; no metrics will be dispatched")
	} else if client, err := NewInfluxDBClient(re); err == nil {
		m.tags = map[string]string{
			"run":      re.TestRun,
			"group_id": re.TestGroupID,
		}

		m.influxdb = client
		if InfluxTestBatcher {
			m.batcher = &nilBatcher{client}
		} else {
			m.batcher = newBatcher(re, client, InfluxBatchLength, InfluxBatchInterval, InfluxBatchRetryOpts(re)...)
		}

		dsinks = append(dsinks, m.writeToInfluxDBSink("diagnostics"))
	} else {
		re.RecordMessage("InfluxDB unavailable; no metrics will be dispatched: %s", err)
	}

	m.diagnostics = newMetricsApi(re, metricsApiOpts{
		freq:        5 * time.Second,
		preregister: metrics.RegisterRuntimeMemStats,
		callbacks:   []func(metrics.Registry){metrics.CaptureRuntimeMemStatsOnce},
		sinks:       dsinks,
	})

	m.results = newMetricsApi(re, metricsApiOpts{
		freq:  1 * time.Second,
		sinks: []MetricSinkFn{m.logSinkJSON("results.out")},
	})

	return m
}

func (m *Metrics) R() *MetricsApi {
	return m.results
}

func (m *Metrics) D() *MetricsApi {
	return m.diagnostics
}

func (m *Metrics) Close() error {
	var err *multierror.Error

	// close diagnostics; this stops the ticker and any further observations on
	// runenv.D() will fail/panic.
	err = multierror.Append(err, m.diagnostics.Close())

	// close results; no more results via runenv.R() can be recorded.
	err = multierror.Append(err, m.results.Close())

	if m.influxdb != nil {
		// Next, we reopen the results.out file, and write all points to InfluxDB.
		results := filepath.Join(m.re.TestOutputsPath, "results.out")
		if file, errf := os.OpenFile(results, os.O_RDONLY, 0666); errf == nil {
			err = multierror.Append(err, m.batchInsertInfluxDB(file))
		} else {
			err = multierror.Append(err, errf)
		}
	}

	// Flush the immediate InfluxDB writer.
	if m.batcher != nil {
		err = multierror.Append(err, m.batcher.Close())
	}

	// Now we're ready to close InfluxDB.
	if m.influxdb != nil {
		err = multierror.Append(err, m.influxdb.Close())
	}

	return err.ErrorOrNil()
}

func (m *Metrics) batchInsertInfluxDB(results *os.File) error {
	sink := m.writeToInfluxDBSink("results")

	for dec := json.NewDecoder(results); dec.More(); {
		var me Metric
		if err := dec.Decode(&me); err != nil {
			m.re.RecordMessage("failed to decode Metric from results.out: %s", err)
			continue
		}

		if err := sink(&me); err != nil {
			m.re.RecordMessage("failed to process Metric from results.out: %s", err)
		}
	}
	return nil
}

func (m *Metrics) logSinkJSON(filename string) MetricSinkFn {
	f, err := m.re.CreateRawAsset(filename)
	if err != nil {
		panic(err)
	}

	enc := json.NewEncoder(f)
	return func(m *Metric) error {
		return enc.Encode(m)
	}
}

func (m *Metrics) computeTags(name string, customtags []string) map[string]string {
	ret := make(map[string]string, len(m.tags)+len(customtags))

	// copy global tags.
	for k, v := range m.tags {
		ret[k] = v
	}

	// process custom tags.
	for _, t := range customtags {
		kv := strings.Split(t, "=")
		if len(kv) != 2 {
			m.re.SLogger().Warnf("skipping invalid tag for metric; name: %s, tag: %s", name, t)
			continue
		}
		ret[kv[0]] = kv[1]
	}
	return ret
}

func (m *Metrics) writeToInfluxDBSink(measurementType string) MetricSinkFn {
	return func(metric *Metric) error {
		fields := make(map[string]interface{}, len(metric.Measures))
		for k, v := range metric.Measures {
			fields[k] = v

			var tags map[string]string
			vals := strings.Split(metric.Name, ",")
			if len(vals) > 1 {
				// we have custom metric tags; inject global tags + provided tags.
				tags = m.computeTags(vals[0], vals[1:])
			} else {
				// we have no custom metric tags; inject global tags only.
				tags = m.tags
			}

			prefix := fmt.Sprintf("%s.%s-%s", measurementType, m.re.TestPlan, m.re.TestCase)
			measurementName := fmt.Sprintf("%s.%s.%s", prefix, vals[0], metric.Type.String())

			p, err := client.NewPoint(measurementName, tags, fields, time.Unix(0, metric.Timestamp))
			if err != nil {
				return err
			}
			m.batcher.WritePoint(p)
		}
		return nil
	}
}

func (m *Metrics) recordEvent(evt *Event) {
	if m.influxdb == nil {
		return
	}

	// this map copy is terrible; the influxdb v2 SDK makes points mutable.
	tags := make(map[string]string, len(m.tags)+2)
	for k, v := range m.tags {
		tags[k] = v
	}

	fields := map[string]interface{}{
		"count": 1,
	}

	tags["event_type"] = evt.Type()

	p, err := client.NewPoint("events", tags, fields)
	if err != nil {
		m.re.RecordMessage("failed to create InfluxDB point: %s", err)
	}

	m.batcher.WritePoint(p)
}
