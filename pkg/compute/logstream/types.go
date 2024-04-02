package logstream

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Streamer is an interface for streaming execution logs through a channel
type Streamer interface {
	// Stream returns a channel of execution logs
	Stream(ctx context.Context) chan *concurrency.AsyncResult[models.ExecutionLog]
}
