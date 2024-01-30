package logstream

import (
	"context"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type StreamParams struct {
	Reader io.Reader
	Buffer int
}

type Stream struct {
	LogChannel chan *models.ExecutionLog
	reader     io.Reader
}

func NewStream(ctx context.Context, params StreamParams) *Stream {
	s := &Stream{
		LogChannel: make(chan *models.ExecutionLog, params.Buffer),
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
				executionLog, err := s.readExecutionLog()
				if err != nil {
					if err != io.EOF {
						// return one last log entry with the error message before closing the channel
						s.LogChannel <- &models.ExecutionLog{
							Error: err.Error(),
						}
					}
					return
				}
				s.LogChannel <- executionLog
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
