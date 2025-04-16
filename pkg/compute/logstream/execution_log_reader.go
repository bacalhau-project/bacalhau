package logstream

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
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
	ctx          context.Context
	isClosed     bool
	readCancelCh chan struct{}
	pipeReader   io.ReadCloser
	params       *ExecutionLogReaderParams
}

func NewReaderForRequest(ctx context.Context, logsDir string, request messages.ExecutionLogsRequest) (*ExecutionLogReader, error) {
	// TODO: This is a legacy way of intepreting the "Tail" parameter.
	// It should be modified to return the last N frames from the log.
	var since time.Time
	if request.Tail {
		since = time.Now()
	} else {
		since = time.Unix(0, 0)
	}

	return &ExecutionLogReader{
		ctx:          ctx,
		readCancelCh: make(chan struct{}, 1),
		params: &ExecutionLogReaderParams{
			logsDir: logsDir,
			since:   since,
			follow:  request.Follow,
		},
	}, nil
}

func (rc *ExecutionLogReader) Close() error {
	// Do nothing if already closed to avoid sending multiple close signals
	if rc.isClosed {
		return nil
	}
	rc.isClosed = true
	rc.readCancelCh <- struct{}{}
	close(rc.readCancelCh)
	return nil
}

func (rc *ExecutionLogReader) Read(p []byte) (n int, err error) {
	// If already closed, return EOF
	if rc.isClosed {
		return 0, io.EOF
	}

	pipedReader, err := rc.getPipeReader()
	if err != nil {
		return 0, err
	}

	return pipedReader.Read(p)
}

func (rc *ExecutionLogReader) initPipe() (*io.PipeReader, *io.PipeWriter, error) {
	// Wait for the log file to be created by the executor
	fileReader, err := rc.logFileWait()
	if err != nil {
		return nil, nil, err
	}

	// Create a pipe that connects filtered logs from the file and the calling function that expects a Reader
	pipeReader, pipeWriter := io.Pipe()

	// Start reading and filtering the logs from the file
	// and writing them to the pipe so the caller can read them
	go rc.startReading(pipeWriter, fileReader)
	return pipeReader, pipeWriter, nil
}

func (rc *ExecutionLogReader) getPipeReader() (io.ReadCloser, error) {
	if rc.pipeReader != nil {
		return rc.pipeReader, nil
	}
	var err error
	rc.pipeReader, _, err = rc.initPipe()
	if err != nil {
		return nil, err
	}
	return rc.pipeReader, nil
}

func (rc *ExecutionLogReader) startReading(pipeWriter io.WriteCloser, logFileReader io.ReadCloser) {
	// Close the pipe writer and log file reader when done
	defer pipeWriter.Close()
	// Close the file reader when done
	defer closer.CloseWithLogOnError("execution_log_file_reader", logFileReader)
	_, err := TimestampedStdCopy(pipeWriter, logFileReader, &rc.params.since, rc.params.follow, rc.readCancelCh)
	if err != nil {
		log.Error().Err(err).Msg("error reading execution log")
	}
}

func (rc *ExecutionLogReader) logFileWait() (io.ReadCloser, error) {
	// Create a context with a timeout to avoid waiting indefinitely
	ctx, cancel := context.WithTimeout(rc.ctx, executionLogFileMaxWaitTime)
	defer cancel()

	// Periodically check for the log file
	ticker := time.NewTicker(executionLogWaitInterval)
	defer ticker.Stop()

	waitStart := time.Now()
	filePath := filepath.Join(rc.params.logsDir, compute.ExecutionLogFileName)
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
		case <-ctx.Done():
			log.Debug().
				Err(ctx.Err()).
				Str("path", filePath).
				Msg("context resolved while waiting for execution logs")
			return nil, ctx.Err()
		case <-ticker.C:
			continue
		}
	}
}
