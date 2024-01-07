package nats

import (
	"github.com/nats-io/nats-server/v2/server"
	"github.com/rs/zerolog"
)

// ZeroLogger is a wrapper around zerolog.Logger to implement the NATS Logger interface
type ZeroLogger struct {
	logger zerolog.Logger
}

// NewZeroLogger creates a new ZeroLogger
func NewZeroLogger(logger zerolog.Logger) ZeroLogger {
	return ZeroLogger{
		logger: logger,
	}
}

func (N ZeroLogger) Noticef(format string, v ...interface{}) {
	N.logger.Info().Msgf(format, v...)
}

func (N ZeroLogger) Warnf(format string, v ...interface{}) {
	N.logger.Warn().Msgf(format, v...)
}

func (N ZeroLogger) Fatalf(format string, v ...interface{}) {
	N.logger.Fatal().Msgf(format, v...)
}

func (N ZeroLogger) Errorf(format string, v ...interface{}) {
	N.logger.Error().Msgf(format, v...)
}

func (N ZeroLogger) Debugf(format string, v ...interface{}) {
	N.logger.Debug().Msgf(format, v...)
}
func (N ZeroLogger) Tracef(format string, v ...interface{}) {
	N.logger.Trace().Msgf(format, v...)
}

// compile-time check whether the ZeroLogger implements the Logger interface
var _ server.Logger = (*ZeroLogger)(nil)
