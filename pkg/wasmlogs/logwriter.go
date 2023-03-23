package wasmlogs

import (
	"context"
	"io"
	"time"
)

type LogWriter struct {
	ctx      context.Context
	stream   string
	ch       chan<- Message
	template Message
}

func NewLogWriter(ctx context.Context, stream string, ch chan<- Message) *LogWriter {
	return &LogWriter{
		ctx:      ctx,
		stream:   stream,
		ch:       ch,
		template: Message{Stream: stream},
	}
}

func (w *LogWriter) Write(p []byte) (int, error) {
	// If we have been canceled, stop writing..
	if w.ctx.Err() != nil {
		return 0, io.ErrClosedPipe
	}

	w.template.Timestamp = time.Now().Unix()
	w.template.Data = append([]byte(nil), p...)
	w.ch <- w.template

	return len(p), nil
}
