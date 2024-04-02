package logstream

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type CompletedStreamerParams struct {
	Execution *models.Execution
}

// CompletedStreamer is a streamer for completed executions that streams the
// output from the execution's RunOutput field to the channel.
type CompletedStreamer struct {
	execution *models.Execution
}

func NewCompletedStreamer(params CompletedStreamerParams) *CompletedStreamer {
	return &CompletedStreamer{
		execution: params.Execution,
	}
}

func (s *CompletedStreamer) Stream(ctx context.Context) chan *concurrency.AsyncResult[models.ExecutionLog] {
	ch := make(chan *concurrency.AsyncResult[models.ExecutionLog])
	go func() {
		defer close(ch)
		if s.execution.RunOutput != nil {
			s.process(ctx, ch, s.execution.RunOutput.STDOUT, models.ExecutionLogTypeSTDOUT)
			s.process(ctx, ch, s.execution.RunOutput.STDERR, models.ExecutionLogTypeSTDERR)
		}
	}()
	return ch
}

func (s *CompletedStreamer) process(
	ctx context.Context, ch chan *concurrency.AsyncResult[models.ExecutionLog], output string, typ models.ExecutionLogType) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			// Context has been cancelled, stop processing
			return
		default:
			asyncResult := concurrency.AsyncResult[models.ExecutionLog]{
				Value: models.ExecutionLog{
					Type: typ,
					Line: scanner.Text() + "\n",
				},
			}
			ch <- &asyncResult
		}
	}

	// Check if context is done before handling scanner errors
	select {
	case <-ctx.Done():
		// If the context is cancelled, ignore scanner errors and exit
		return
	default:
		// If the context is not cancelled, handle scanner errors
		if err := scanner.Err(); err != nil {
			asyncResult := concurrency.AsyncResult[models.ExecutionLog]{
				Err: fmt.Errorf("failed to read output: %w", scanner.Err()),
			}
			ch <- &asyncResult
		}
	}
}

// compile time check
var _ Streamer = (*CompletedStreamer)(nil)
