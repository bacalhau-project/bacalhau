package logstream

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

type ServerParams struct {
	ExecutionStore store.ExecutionStore
	Executors      executor.ExecutorProvider
	Buffer         int
}

type server struct {
	executionStore store.ExecutionStore
	executors      executor.ExecutorProvider
	buffer         int
}

// NewServer creates a new log stream server
func NewServer(params ServerParams) Server {
	return &server{
		executionStore: params.ExecutionStore,
		executors:      params.Executors,
		buffer:         params.Buffer,
	}
}

// GetLogStream returns a stream of logs for a given execution
func (s *server) GetLogStream(ctx context.Context, request requests.LogStreamRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return nil, err
	}

	if execution.IsTerminalComputeState() {
		return nil, fmt.Errorf("can't stream logs for completed execution: %s", request.ExecutionID)
	}
	engineType := execution.Job.Task().Engine.Type
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

// compile time check
var _ Server = &server{}
