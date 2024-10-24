package logstream

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// defaultBuffer is the default size of the channel buffer for each individual log stream.
// A buffer of 100 provides a good balance between memory usage and performance:
// - It's large enough to handle bursts of log messages without blocking the producer.
// - It's small enough to avoid excessive memory usage for long-running streams.
// - This value can be adjusted based on expected log volume and system resources.
const defaultBuffer = 100

type ServerParams struct {
	ExecutionStore store.ExecutionStore
	Executors      executor.ExecProvider
	// Buffer is the size of the channel buffer for each individual log stream.
	// If not set (0), defaultBuffer will be used.
	Buffer int
}

type Server struct {
	executionStore store.ExecutionStore
	executors      executor.ExecProvider
	// buffer is the size of the channel buffer for each individual log stream.
	buffer int
}

// NewServer creates a new log stream server
func NewServer(params ServerParams) *Server {
	if params.Buffer <= 0 {
		params.Buffer = defaultBuffer
	}
	return &Server{
		executionStore: params.ExecutionStore,
		executors:      params.Executors,
		buffer:         params.Buffer,
	}
}

// GetLogStream returns a stream of logs for a given execution
func (s *Server) GetLogStream(ctx context.Context, request executor.LogStreamRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	localExecutionState, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return nil, err
	}

	if localExecutionState.State.IsTerminal() {
		return nil, fmt.Errorf("can't stream logs for completed execution: %s", request.ExecutionID)
	}
	engineType := localExecutionState.Execution.Job.Task().Engine.Type
	exec, err := s.executors.Get(ctx, engineType)
	if err != nil {
		return nil, fmt.Errorf("failed to find executor for engine: %s. %w", engineType, err)
	}

	reader, err := exec.GetLogStream(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get log stream for execution: %s. %w", request.ExecutionID, err)
	}
	streamer := NewLiveStreamer(LiveStreamerParams{
		Reader: reader,
		Buffer: s.buffer,
	})

	return streamer.Stream(ctx), nil
}
