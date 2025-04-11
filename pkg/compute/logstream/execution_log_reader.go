package logstream

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
)

const (
	executionLogFileMaxWaitTime = 5 * time.Second
	executionLogWaitInterval    = 100 * time.Millisecond
)

type ExecutionLogReaderParams struct {
	since   time.Time
	follow  bool
	logsDir string
}

type ExecutionLogReader struct {
	params *ExecutionLogReaderParams
}

func NewExecutionLogReader(params *ExecutionLogReaderParams) (*ExecutionLogReader, error) {
	// Execution logs directory is expected to exist. If it doesn't, return an error
	if _, err := os.Stat(params.logsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("execution logs directory does not exist: %s", params.logsDir)
	}

	return &ExecutionLogReader{params}, nil
}

// Creates a reader for the logs of a given execution. The logs are served from the local file(s)
// captured during execution run.
// It is possible that the execution has not yet created any log files,
// in which case the reader will make a best-effort attemt to wait for the log files to appear.
func NewExecutionLogReaderFromRequest(logsDir string, request messages.ExecutionLogsRequest) (
	*ExecutionLogReader,
	error,
) {
	// TODO: This is a legacy way of intepreting the "Tail" parameter.
	// It should ideally return the last N frames from the logs.
	var since time.Time
	if request.Tail {
		since = time.Now()
	} else {
		since = time.Unix(0, 0)
	}

	return NewExecutionLogReader(&ExecutionLogReaderParams{since, request.Follow, logsDir})
}

// Asynchronously read execution logs to the given destination.
// This function will wait until the log files are created before starting to read.
// This function will attempt to cancel the reading if notified via the cancelCh channel.
func (r *ExecutionLogReader) StartReading(dst io.Writer, cancelCh chan struct{}) chan util.Result[int64] {
	resultCh := make(chan util.Result[int64])
	go func() {
		defer close(resultCh)
		src, err := fileWait(r.params.logsDir)
		if err != nil {
			resultCh <- util.NewResult[int64](0, err)
			return
		}

		defer closer.CloseWithLogOnError("executionLogReader", src)
		copiedBytes, err := TimestampedStdCopy(dst, src, &r.params.since, r.params.follow, cancelCh)
		select {
		case resultCh <- util.NewResult(copiedBytes, err):
		default:
		}
	}()
	return resultCh
}

// TODO: This should be aware of log storage schema: rotation, file naming format, etc.
// TODO: Error messages from this function are exposed to the user, they should be more user-friendly.
// Returns a reader for raw execution logs. The caller is responsible for closing the reader when done.
func fileWait(logsDir string) (io.ReadCloser, error) {
	filePath := filepath.Join(logsDir, compute.ExecutionLogFileName)
	waitStart := time.Now()
	timeout := time.After(executionLogFileMaxWaitTime)
	ticker := time.NewTicker(executionLogWaitInterval)
	defer ticker.Stop()

	for {
		file, err := os.Open(filePath)
		if err == nil {
			log.Debug().
				Str("path", filePath).
				Str("wait_time", time.Since(waitStart).String()).
				Msg("execution log file found")
			return file, nil
		}

		if !os.IsNotExist(err) {
			msg := "error opening log file"
			log.Error().Str("path", filePath).Err(err).Msg(msg)
			return nil, errors.New(msg)
		}

		select {
		case <-timeout:
			msg := "timeout while waiting for execution logs"
			log.Debug().Str("path", filePath).Msg(msg)
			return nil, errors.New(msg)
		case <-ticker.C:
			continue
		}
	}
}
