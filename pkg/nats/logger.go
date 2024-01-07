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

func (N ZeroLogger) Noticef(format string, v ...interface{}) {
	N.logWithLevel(zerolog.InfoLevel, format, v)
}

func (N ZeroLogger) Warnf(format string, v ...interface{}) {
	N.logWithLevel(zerolog.WarnLevel, format, v)
}

func (N ZeroLogger) Fatalf(format string, v ...interface{}) {
	N.logWithLevel(zerolog.FatalLevel, format, v)
}

func (N ZeroLogger) Errorf(format string, v ...interface{}) {
	N.logWithLevel(zerolog.ErrorLevel, format, v)
}

func (N ZeroLogger) Debugf(format string, v ...interface{}) {
	N.logWithLevel(zerolog.DebugLevel, format, v)
}

func (N ZeroLogger) Tracef(format string, v ...interface{}) {
	N.logWithLevel(zerolog.TraceLevel, format, v)
}

func (N ZeroLogger) logWithLevel(level zerolog.Level, format string, v []interface{}) {
	N.logger.WithLevel(level).Str("Server", N.serverID).Msgf(format, v...)
}

// compile-time check whether the ZeroLogger implements the Logger interface
var _ server.Logger = (*ZeroLogger)(nil)
