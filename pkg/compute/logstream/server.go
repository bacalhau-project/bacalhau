package logstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
)

const (
	// defaultBuffer is the default size of the channel buffer for each individual log stream.
	// A buffer of 100 provides a good balance between memory usage and performance:
	// - It's large enough to handle bursts of log messages without blocking the producer.
	// - It's small enough to avoid excessive memory usage for long-running streams.
	// - This value can be adjusted based on expected log volume and system resources.
	defaultBuffer = 100

	// How long to wait for an execution to appear in the store before giving up
	executionWaitTimeout = 5 * time.Second

	// How often to check for the execution in the store
	executionWaitInterval = 100 * time.Millisecond
)

type ServerParams struct {
	ExecutionStore store.ExecutionStore
	Executors      executor.ExecProvider
	Buffer         int // If not set (0), defaultBuffer will be used.
	ResultsPath    compute.ResultsPath
}

type server struct {
	executionStore store.ExecutionStore
	buffer         int
	resultsPath    compute.ResultsPath
}

// NewServer creates a new log stream server
func NewServer(params ServerParams) Server {
	if params.Buffer <= 0 {
		params.Buffer = defaultBuffer
	}

	return &server{
		executionStore: params.ExecutionStore,
		buffer:         params.Buffer,
		resultsPath:    params.ResultsPath,
	}
}

// GetLogStream returns a stream of logs for a given execution
func (s *server) GetLogStream(ctx context.Context, request messages.ExecutionLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	log.Debug().Str("execution", request.ExecutionID).Msg("creating log stream")

	execution, err := s.executionWait(ctx, request.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("execution %s not found", request.ExecutionID)
	}

	logsDir := compute.ExecutionLogsDir(s.resultsPath.ExecutionOutputDir(execution.ID))
	outputReader, err := NewExecutionLogReaderFromRequest(logsDir, request)
	if err != nil {
		return nil, err
	}

	cancelCh := make(chan struct{})
	reader, writer := io.Pipe()
	readingResultCh := outputReader.StartReading(writer, cancelCh)
	go func() {
		readingResult := <-readingResultCh
		defer closer.CloseWithLogOnError("execution_stream_pipe", writer)
		if readingResult.Error != nil {
			log.Error().Err(err).Msg("execution log reader failed")
		} else {
			log.Debug().
				Str("execution", execution.ID).
				Int64("size_bytes", readingResult.Value).
				Msg("execution log reader finished")
		}
	}()

	// Cancel the reader when the context is done. This is a best effort attempt,
	// as the reader may have already been closed by the time the context is done.
	go func() {
		<-ctx.Done()
		log.Trace().Str("execution", execution.ID).Msg("attempting to cancel log reader")
		select {
		case cancelCh <- struct{}{}:
			// Cancel reader
			log.Trace().Str("execution", execution.ID).Msg("log reader cancelled")
		default:
			// Channel is full or no receiver, skip it to avoid blocking
			log.Trace().Str("execution", execution.ID).Msg("log reader cancel skipped")
		}
	}()

	streamer := NewLiveStreamer(LiveStreamerParams{
		Reader: reader,
		Buffer: s.buffer,
	})

	return streamer.Stream(ctx), nil
}

// Best-effort attempt to find an execution with the given ID. A request for logs can arrive before the execution
// is actually created or it still sits in the buffer (see ExecutorBuffer).
// This method will wait for a short amount of time and return nil if the execution is not found.
func (s *server) executionWait(ctx context.Context, executionID string) (*models.Execution, error) {
	waitStart := time.Now()
	timeout := time.After(executionWaitTimeout)
	ticker := time.NewTicker(executionWaitInterval)
	defer ticker.Stop()

	for {
		execution, err := s.executionStore.GetExecution(ctx, executionID)
		if err == nil {
			log.Debug().
				Str("execution", execution.ID).
				Str("wait_time", time.Since(waitStart).String()).
				Msg("execution found")
			return execution, nil
		} else if !errors.As(err, &store.ErrExecutionNotFound{}) {
			// If the error is not "not found", return it
			return nil, err
		}

		select {
		case <-ctx.Done():
			msg := "cancelled while waiting for execution"
			log.Debug().Str("execution", executionID).Msg(msg)
			return nil, errors.New(msg)
		case <-timeout:
			msg := "timeout while waiting for execution"
			log.Debug().Str("execution", executionID).Msg(msg)
			return nil, errors.New(msg)
		case <-ticker.C:
			continue
		}
	}
}

// compile time check
var _ Server = &server{}
