package util

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
)

// LogDebugIfContextCanceled will ensure that LOG_LEVEL is set to debug if
// the context is canceled.
func LogDebugIfContextCancelled(ctx context.Context, cleanupErr error, msg string) {
	if cleanupErr == nil {
		return
	}
	if !errors.Is(cleanupErr, context.Canceled) {
		log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to close " + msg)
	} else {
		log.Ctx(ctx).Debug().Err(cleanupErr).Msgf("Context canceled: %s", msg)
	}
}
