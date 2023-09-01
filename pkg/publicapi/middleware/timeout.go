package middleware

import (
	"net/http"
	"time"
)

const DefaultTimeoutMessage = "Server Timeout!"

type TimeoutConfig struct {
	Timeout      time.Duration
	Message      string
	SkippedPaths []string
}

type TimeoutHandler struct {
	timeout        time.Duration
	message        string
	skippedPaths   map[string]struct{}
	nextHandler    http.Handler
	timeoutHandler http.Handler
}

func newTimeoutHandler(config TimeoutConfig, next http.Handler) *TimeoutHandler {
	skippedPaths := make(map[string]struct{})
	for _, path := range config.SkippedPaths {
		skippedPaths[path] = struct{}{}
	}
	return &TimeoutHandler{
		timeout:        config.Timeout,
		message:        config.Message,
		skippedPaths:   skippedPaths,
		nextHandler:    next,
		timeoutHandler: http.TimeoutHandler(next, config.Timeout, config.Message),
	}
}

func (h *TimeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.skippedPaths[r.URL.Path]; ok {
		h.nextHandler.ServeHTTP(w, r)
		return
	}
	h.timeoutHandler.ServeHTTP(w, r)
}

// Timeout is a middleware to add http.TimeoutHandler.
func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return TimeoutWithConfig(TimeoutConfig{
		Timeout: timeout,
		Message: DefaultTimeoutMessage,
	})
}

// TimeoutWithConfig is a middleware to add http.TimeoutHandler with custom message.
func TimeoutWithConfig(config TimeoutConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return newTimeoutHandler(config, next)
	}
}
