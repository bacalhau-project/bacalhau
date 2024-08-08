package handlers

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

type LoggingHandler struct {
	logger zerolog.Logger
}

func NewLoggingHandler(
	logger zerolog.Logger,
) *LoggingHandler {
	return &LoggingHandler{
		logger: logger,
	}
}

func (p *LoggingHandler) HandleEvent(ctx context.Context, event watcher.Event) error {
	p.logger.Info().Ctx(ctx).
		Str("event_type", event.ObjectType).
		Dur("event_age", time.Since(event.Timestamp)).
		Msgf("Processing event: %+v", event)
	return nil
}
