package closer

import (
	"errors"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// CloseWithLogOnError will close the given resource and log any relevant failure
func CloseWithLogOnError(name string, c io.Closer) {
	err := c.Close()
	if err == nil || errors.Is(err, os.ErrClosed) {
		return
	}

	l := log.With().CallerWithSkipFrameCount(3).Logger()
	l.Err(err).Msgf("Failed to close %s", name)
}
