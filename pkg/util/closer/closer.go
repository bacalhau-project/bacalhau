package closer

import (
	"errors"
	"io"
	"net"
	"os"

	"github.com/rs/zerolog/log"
)

// CloseWithLogOnError will close the given resource and log any relevant failure
func CloseWithLogOnError(name string, c io.Closer) {
	err := c.Close()
	if err == nil || errors.Is(err, os.ErrClosed) || errors.Is(err, net.ErrClosed) {
		return
	}

	l := log.With().CallerWithSkipFrameCount(3).Logger()
	l.Err(err).Msgf("Failed to close %s", name)
}
