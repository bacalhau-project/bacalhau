package middleware

import (
	"net/http"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ZeroLogFormatter struct {
	logger   zerolog.Logger
	logLevel zerolog.Level
}

type ZeroLogFormatterOption func(*ZeroLogFormatter)

// WithLogger sets a logger for the ZeroLogFormatter.
func WithLogger(logger zerolog.Logger) ZeroLogFormatterOption {
	return func(z *ZeroLogFormatter) {
		z.logger = logger
	}
}

// WithLogLevel sets a log level for the ZeroLogFormatter.
func WithLogLevel(logLevel zerolog.Level) ZeroLogFormatterOption {
	return func(z *ZeroLogFormatter) {
		z.logLevel = logLevel
	}
}

// NewZeroLogFormatter returns a new ZeroLogFormatter configured with the provided option setters.
func NewZeroLogFormatter(options ...ZeroLogFormatterOption) *ZeroLogFormatter {
	// default values
	formatter := &ZeroLogFormatter{
		logger:   log.Logger,
		logLevel: zerolog.InfoLevel,
	}

	// apply the options
	for _, option := range options {
		option(formatter)
	}

	return formatter
}

// NewLogEntry returns a new LogEntry for the request.
func (l *ZeroLogFormatter) NewLogEntry(r *http.Request) chimiddleware.LogEntry {
	return zeroLogEntry{
		request:   r,
		formatter: l,
	}
}

type zeroLogEntry struct {
	request   *http.Request
	formatter *ZeroLogFormatter
}

// Write implements the io.Writer interface to write a string to the logger.
func (l zeroLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	logLevel := l.formatter.logLevel
	if status >= http.StatusInternalServerError && logLevel < zerolog.ErrorLevel {
		logLevel = zerolog.ErrorLevel
	} else if status >= http.StatusBadRequest && logLevel < zerolog.WarnLevel {
		logLevel = zerolog.WarnLevel
	}
	l.formatter.logger.WithLevel(logLevel).
		Str("Method", l.request.Method).
		Str("URI", l.request.URL.String()).
		Str("RemoteAddr", l.request.RemoteAddr).
		Int("StatusCode", status).
		Int("Size", bytes).
		Dur("Duration", elapsed).
		Str("Referer", l.request.Referer()).
		Str("UserAgent", l.request.UserAgent()).
		Str("ClientID", header.Get(apimodels.HTTPHeaderClientID)).
		Str("JobID", header.Get(apimodels.HTTPHeaderJobID)).
		Send()
}

// Panic implements the LogEntry interface to log a panic occurred during the request.
func (l zeroLogEntry) Panic(v interface{}, stack []byte) {
	l.formatter.logger.Error().Bytes("stack", stack).Msgf("Panic: %v", v)
}
