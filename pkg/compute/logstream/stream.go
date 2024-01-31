package logstream

import (
	"context"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type StreamParams struct {
	Reader io.Reader
	Buffer int
}

type Stream struct {
	LogChannel chan *concurrency.AsyncResult[models.ExecutionLog]
	reader     io.Reader
}

func NewStream(ctx context.Context, params StreamParams) *Stream {
	s := &Stream{
		LogChannel: make(chan *concurrency.AsyncResult[models.ExecutionLog], params.Buffer),
		reader:     params.Reader,
	}
	s.start(ctx)
	return s
}

func (s *Stream) start(ctx context.Context) {
	go func() {
		defer close(s.LogChannel)
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
						s.LogChannel <- asyncResult
					}
					return
				}
				asyncResult.Value = *executionLog
				s.LogChannel <- asyncResult
			}
		}
	}()
}

func (s *Stream) readExecutionLog() (*models.ExecutionLog, error) {
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
