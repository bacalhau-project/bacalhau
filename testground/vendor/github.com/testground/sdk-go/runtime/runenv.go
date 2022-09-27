package runtime

import (
	"context"
	"os"
	gosync "sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/hashicorp/go-multierror"
	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	"go.uber.org/zap"
)

var (
	InfluxTestBatcher    = true // used for testing purposes
	InfluxBatchLength    = 128
	InfluxBatchInterval  = 1 * time.Second
	InfluxBatchRetryOpts = func(re *RunEnv) []retry.Option {
		return []retry.Option{
			retry.Attempts(5),
			retry.Delay(500 * time.Millisecond),
			retry.OnRetry(func(n uint, err error) {
				re.RecordMessage("failed to send batch to InfluxDB; attempt %d; err: %s", n, err)
			}),
		}
	}
)

// RunEnv encapsulates the context for this test run.
type RunEnv struct {
	RunParams

	logger        *zap.Logger
	metrics       *Metrics
	signalEmitter SignalEmitter

	wg        gosync.WaitGroup
	closeCh   chan struct{}
	assetsErr error

	unstructured struct {
		files []*os.File
		ch    chan *os.File
	}
	structured struct {
		loggers []*zap.Logger
		ch      chan *zap.Logger
	}
}

func (re *RunEnv) SLogger() *zap.SugaredLogger {
	return re.logger.Sugar()
}

// NewRunEnv constructs a runtime environment from the given runtime parameters.
func NewRunEnv(params RunParams) *RunEnv {
	re := &RunEnv{
		RunParams: params,
		closeCh:   make(chan struct{}),
	}
	re.initLogger()

	re.structured.ch = make(chan *zap.Logger)
	re.unstructured.ch = make(chan *os.File)
	re.signalEmitter = &NilSignalEmitter{}

	re.wg.Add(1)
	go re.manageAssets()

	re.metrics = newMetrics(re)

	return re
}

type SignalEmitter interface {
	SignalEvent(context.Context, *Event) error
}

type NilSignalEmitter struct{}

func (ne NilSignalEmitter) SignalEvent(ctx context.Context, event *Event) error {
	return nil
}

func (re *RunEnv) AttachSyncClient(se SignalEmitter) {
	re.signalEmitter = se
}

// R returns a metrics object for results.
func (re *RunEnv) R() *MetricsApi {
	return re.metrics.R()
}

// D returns a metrics object for diagnostics.
func (re *RunEnv) D() *MetricsApi {
	return re.metrics.D()
}

func (re *RunEnv) manageAssets() {
	defer re.wg.Done()

	var err *multierror.Error
	defer func() { re.assetsErr = err.ErrorOrNil() }()

	for {
		select {
		case f := <-re.unstructured.ch:
			re.unstructured.files = append(re.unstructured.files, f)
		case l := <-re.structured.ch:
			re.structured.loggers = append(re.structured.loggers, l)
		case <-re.closeCh:
			for _, f := range re.unstructured.files {
				err = multierror.Append(err, f.Close())
			}
			for _, l := range re.structured.loggers {
				err = multierror.Append(err, l.Sync())
			}
			return
		}
	}
}

func (re *RunEnv) Close() error {
	var err *multierror.Error

	// close metrics.
	err = multierror.Append(err, re.metrics.Close())

	// This close stops monitoring the wapi errors channel, and closes assets.
	close(re.closeCh)
	re.wg.Wait()
	err = multierror.Append(err, re.assetsErr)

	if l := re.logger; l != nil {
		_ = l.Sync()
	}

	return err.ErrorOrNil()
}

// CurrentRunEnv populates a test context from environment vars.
func CurrentRunEnv() *RunEnv {
	re, _ := ParseRunEnv(os.Environ())
	return re
}

// ParseRunEnv parses a list of environment variables into a RunEnv.
func ParseRunEnv(env []string) (*RunEnv, error) {
	p, err := ParseRunParams(env)
	if err != nil {
		return nil, err
	}

	return NewRunEnv(*p), nil
}
