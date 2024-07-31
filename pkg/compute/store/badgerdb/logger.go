package badgerdb

import (
	"github.com/rs/zerolog"
)

type badgerLoggerAdapter struct {
	logger   zerolog.Logger
	minLevel zerolog.Level
}

func newBadgerLoggerAdapter(logger zerolog.Logger, minLevel zerolog.Level) *badgerLoggerAdapter {
	return &badgerLoggerAdapter{
		logger:   logger,
		minLevel: minLevel,
	}
}

func (l *badgerLoggerAdapter) Errorf(f string, v ...interface{}) {
	if l.minLevel <= zerolog.ErrorLevel {
		l.logger.Error().Msgf(f, v...)
	}
}

func (l *badgerLoggerAdapter) Warningf(f string, v ...interface{}) {
	if l.minLevel <= zerolog.WarnLevel {
		l.logger.Warn().Msgf(f, v...)
	}
}

func (l *badgerLoggerAdapter) Infof(f string, v ...interface{}) {
	if l.minLevel <= zerolog.InfoLevel {
		l.logger.Info().Msgf(f, v...)
	}
}

func (l *badgerLoggerAdapter) Debugf(f string, v ...interface{}) {
	if l.minLevel <= zerolog.DebugLevel {
		l.logger.Debug().Msgf(f, v...)
	}
}
