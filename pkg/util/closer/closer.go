package closer

import (
	"context"
	"errors"
	"io"
	"net"
	"os"

	"github.com/rs/zerolog/log"
)

// CloseWithLogOnError will close the given resource and log any relevant failure
func CloseWithLogOnError(name string, c io.Closer) {
	closeCloser(name, c)
}

// ContextCloserWithLogOnError will close the given resource using the context and log any relevant failure
func ContextCloserWithLogOnError(ctx context.Context, name string, c CloseWithContext) {
	closeCloser(name, contextIoCloser{ctx, c.Close})
}

// DrainAndCloseWithLogOnError will first ensure the contents of the reader has been read before being closed. This is
// useful when dealing with HTTP response bodies which need to be drained and closed so that the connection may be
// reused by the OS.
func DrainAndCloseWithLogOnError(ctx context.Context, name string, c io.ReadCloser) {
	if _, err := io.Copy(io.Discard, c); err != nil {
		l := log.Ctx(ctx).With().CallerWithSkipFrameCount(3).Logger()
		l.Err(err).Msgf("Failed to drain %s", name)
	}

	closeCloser(name, c)
}

func closeCloser(name string, c io.Closer) {
	err := c.Close()
	if err == nil || errors.Is(err, os.ErrClosed) || errors.Is(err, net.ErrClosed) {
		return
	}

	l := log.With().CallerWithSkipFrameCount(framesFromCaller).Logger()
	l.Err(err).Msgf("Failed to close %s", name)
}

const framesFromCaller = 4

type CloseWithContext interface {
	Close(ctx context.Context) error
}

type contextIoCloser struct {
	ctx context.Context
	f   func(context.Context) error
}

func (c contextIoCloser) Close() error {
	return c.f(c.ctx)
}
