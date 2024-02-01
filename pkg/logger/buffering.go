package logger

import (
	"io"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

var logBufferedLogs func(io.Writer) error

// LogBufferedLogs will log any log messages written from before logging was configured to the given writer. If writer
// is nil, then the default logging will be used instead. This function will do nothing once the buffer has been outputted.
func LogBufferedLogs(writer io.Writer) {
	if logBufferedLogs == nil {
		return
	}
	if writer == nil {
		writer = defaultLogging()
	}

	if err := logBufferedLogs(writer); err != nil {
		log.Err(err).Msg("Failed to log messages")
	}
	logBufferedLogs = nil
}

// bufferLogs is an io.Writer to be used with zerolog which will buffer log messages until LogBufferedLogs is called with
// the real log writer, such as from defaultLogging or jsonLogging.
func bufferLogs() io.Writer {
	buffer := &bufferingLogWriter{}
	logBufferedLogs = buffer.writeLogs
	return buffer
}

type bufferingLogWriter struct {
	buffer [][]byte
	mu     sync.Mutex
}

func (b *bufferingLogWriter) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// make sure p isn't reused while it's being kept on the buffer
	p = slices.Clone(p)

	b.buffer = append(b.buffer, p)

	return 0, nil
}

func (b *bufferingLogWriter) writeLogs(w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var errs error
	for _, line := range b.buffer {
		if _, err := w.Write(line); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

var _ io.Writer = &bufferingLogWriter{}
