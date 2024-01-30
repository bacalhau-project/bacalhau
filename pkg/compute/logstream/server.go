package logstream

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ServerParams struct {
	ExecutionStore store.ExecutionStore
	Executors      executor.ExecutorProvider
}

type Server struct {
	executionStore store.ExecutionStore
	executors      executor.ExecutorProvider
}

// NewServer creates a new log stream server
func NewServer(params ServerParams) *Server {
	return &Server{
		executionStore: params.ExecutionStore,
		executors:      params.Executors,
	}
}

// GetLogStream returns a stream of logs for a given execution
func (s *Server) GetLogStream(ctx context.Context, request executor.LogStreamRequest) (<-chan *models.ExecutionLog, error) {
	localExecutionState, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return nil, err
	}

	if localExecutionState.State.IsTerminal() {
		// TODO: support streaming logs for completed executions
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
	stream := NewStream(ctx, StreamParams{
		Reader: reader,
	})
	return stream.LogChannel, nil
}
