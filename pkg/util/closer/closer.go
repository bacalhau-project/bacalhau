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

// DrainAndCloseWithLogOnError will first ensure the contents of the reader has been read before being closed. This is
// useful when dealing with HTTP response bodies which need to be drained and closed so that the connection may be
// re-used by the OS.
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
