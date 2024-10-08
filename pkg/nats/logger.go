package nats

import (
	"github.com/nats-io/nats-server/v2/server"
	"github.com/rs/zerolog"
)

// ZeroLogger is a wrapper around zerolog.Logger to implement the NATS Logger interface
type ZeroLogger struct {
	logger   zerolog.Logger
	serverID string
}

// NewZeroLogger creates a new ZeroLogger
func NewZeroLogger(logger zerolog.Logger, serverID string) ZeroLogger {
	return ZeroLogger{
		logger:   logger,
		serverID: serverID,
	}
}

func (l ZeroLogger) Noticef(format string, v ...interface{}) {
	// As we are mainly interested in error and warn logs from nats,
	// we set trace level as nats notice/info logs are noisy.
	l.logWithLevel(zerolog.TraceLevel, format, v)
}

func (l ZeroLogger) Warnf(format string, v ...interface{}) {
	l.logWithLevel(zerolog.WarnLevel, format, v)
}

func (l ZeroLogger) Fatalf(format string, v ...interface{}) {
	l.logWithLevel(zerolog.FatalLevel, format, v)
}

func (l ZeroLogger) Errorf(format string, v ...interface{}) {
	l.logWithLevel(zerolog.ErrorLevel, format, v)
}

func (l ZeroLogger) Debugf(format string, v ...interface{}) {
	// Nats debug logs are too noisy, we mark them as trace level instead
	l.logWithLevel(zerolog.TraceLevel, format, v)
}

func (l ZeroLogger) Tracef(format string, v ...interface{}) {
	l.logWithLevel(zerolog.TraceLevel, format, v)
}

func (l ZeroLogger) logWithLevel(level zerolog.Level, format string, v []interface{}) {
	l.logger.WithLevel(level).Str("Server", l.serverID).Msgf(format, v...)
}

// compile-time check whether the ZeroLogger implements the Logger interface
var _ server.Logger = (*ZeroLogger)(nil)
