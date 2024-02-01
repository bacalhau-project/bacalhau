package logstream

import (
	"context"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type LiveStreamerParams struct {
	Reader io.Reader
	Buffer int
}

// LiveStreamer streams logs from a live execution's log stream to a channel
type LiveStreamer struct {
	reader io.Reader
	buffer int
}

func NewLiveStreamer(params LiveStreamerParams) *LiveStreamer {
	return &LiveStreamer{
		reader: params.Reader,
		buffer: params.Buffer,
	}
}

func (s *LiveStreamer) Stream(ctx context.Context) chan *concurrency.AsyncResult[models.ExecutionLog] {
	ch := make(chan *concurrency.AsyncResult[models.ExecutionLog], s.buffer)

	go func() {
		defer close(ch)
		defer func() {
			// close the reader if it's a ReadCloser
			if rc, ok := s.reader.(io.ReadCloser); ok {
				_ = rc.Close()
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				asyncResult := new(concurrency.AsyncResult[models.ExecutionLog])
				executionLog, err := s.readExecutionLog()
				if err != nil {
					if err != io.EOF {
						// return one last log entry with the error message before closing the channel
						asyncResult.Err = err
						ch <- asyncResult
					}
					return
				}
				asyncResult.Value = *executionLog
				ch <- asyncResult
			}
		}
	}()
	return ch
}

func (s *LiveStreamer) readExecutionLog() (*models.ExecutionLog, error) {
	df, err := logger.NewDataFrameFromReader(s.reader)
	if err != nil {
		return nil, err
	}
	logType := models.ExecutionLogTypeSTDERR
	if df.Tag == logger.StdoutStreamTag {
		logType = models.ExecutionLogTypeSTDOUT
	}
	return &models.ExecutionLog{
		Type: logType,
		Line: string(df.Data),
	}, nil
}

// compile-time check that LiveStreamer implements Streamer
var _ Streamer = (*LiveStreamer)(nil)
