package analytics

import (
	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog/log"
)

// ZeroLogger implements the posthog.Logger interface using zerolog
// and logs only failures at debug level
type ZeroLogger struct{}

// NewZeroLogger creates a new ZeroLogger that implements posthog.Logger
func NewZeroLogger() posthog.Logger {
	return ZeroLogger{}
}

// Logf implements the posthog.Logger.Logf method.
// This method intentionally does nothing to suppress all regular logs
func (l ZeroLogger) Logf(format string, args ...interface{}) {
	log.Trace().Msgf("PostHog log: "+format, args...)
}

// Errorf implements the posthog.Logger.Errorf method.
// This logs errors at debug level instead of error level
func (l ZeroLogger) Errorf(format string, args ...interface{}) {
	log.Debug().Msgf("PostHog error: "+format, args...)
}

// Debugf implements the posthog.Logger.Debugf method.
// This logs debug messages at trace level
func (l ZeroLogger) Debugf(format string, args ...interface{}) {
	log.Trace().Msgf("PostHog debug: "+format, args...)
}

// Warnf implements the posthog.Logger.Warnf method.
// This logs warnings at debug level
func (l ZeroLogger) Warnf(format string, args ...interface{}) {
	log.Debug().Msgf("PostHog warning: "+format, args...)
}
