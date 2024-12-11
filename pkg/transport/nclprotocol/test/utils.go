package test

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

// MockLogStreamServer implements a minimal logstream.Server for testing
type MockLogStreamServer struct{}

func (m *MockLogStreamServer) GetLogStream(ctx context.Context, request messages.ExecutionLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	ch := make(chan *concurrency.AsyncResult[models.ExecutionLog])
	close(ch)
	return ch, nil
}
