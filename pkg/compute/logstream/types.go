package logstream

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

// Server is an interface for streaming execution logs
type Server interface {
	// GetLogStream returns a stream of logs for a given execution
	GetLogStream(ctx context.Context, request requests.LogStreamRequest) (
		<-chan *concurrency.AsyncResult[models.ExecutionLog], error)
}

// Streamer is an interface for streaming execution logs through a channel
type Streamer interface {
	// Stream returns a channel of execution logs
	Stream(ctx context.Context) chan *concurrency.AsyncResult[models.ExecutionLog]
}
