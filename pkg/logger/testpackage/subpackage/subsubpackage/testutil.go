package subsubpackage

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// TestLog exists to allow a test to make sure that we don't strip off package names.
func TestLog(errorMessage string, message string) {
	log.Ctx(context.Background()).Err(errors.WithStack(errors.New(errorMessage))).Msg(message)
}
