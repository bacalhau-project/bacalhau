package wasmlogs

import "github.com/bacalhau-project/bacalhau/pkg/util/generic"

type LogWriterTransform func([]byte) *LogMessage

type LogWriter struct {
	buffer      *generic.RingBuffer[*LogMessage]
	transformer LogWriterTransform
}

func NewLogWriter(
	buffer *generic.RingBuffer[*LogMessage],
	transformer LogWriterTransform,
) *LogWriter {
	return &LogWriter{
		buffer:      buffer,
		transformer: transformer,
	}
}

func (w *LogWriter) Write(b []byte) (int, error) {
	transformed := w.transformer(b)
	w.buffer.Enqueue(transformed)
	return len(b), nil
}

func (w *LogWriter) Close() error {
	return nil
}
