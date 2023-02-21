package sensors

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type LoggingSensorParams struct {
	InfoProvider model.DebugInfoProvider
	Interval     time.Duration
}

// LoggingSensor is a sensor that periodically logs the debug info
type LoggingSensor struct {
	infoProvider model.DebugInfoProvider
	interval     time.Duration
}

// NewLoggingSensor create a new LoggingSensor from LoggingSensorParams
func NewLoggingSensor(params LoggingSensorParams) *LoggingSensor {
	return &LoggingSensor{
		infoProvider: params.InfoProvider,
		interval:     params.Interval,
	}
}

func (s LoggingSensor) Start(ctx context.Context) {
	log.Ctx(ctx).Debug().Msgf("starting new logging sensor with interval %s", s.interval)
	ticker := time.NewTicker(s.interval)

	for {
		select {
		case <-ticker.C:
			s.sense(ctx)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (s LoggingSensor) sense(ctx context.Context) {
	debugInfo, err := s.infoProvider.GetDebugInfo()
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to marshal execution summaries")
	} else {
		log.Ctx(ctx).Info().Msgf("%s: %s", debugInfo.Component, debugInfo.Info)
	}
}
