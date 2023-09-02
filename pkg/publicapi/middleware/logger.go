package middleware

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func RequestLogger(logger zerolog.Logger, logLevel zerolog.Level) echo.MiddlewareFunc {
	return echomiddelware.RequestLoggerWithConfig(echomiddelware.RequestLoggerConfig{
		LogMethod:       true,
		LogURI:          true,
		LogRemoteIP:     true,
		LogStatus:       true,
		LogResponseSize: true,
		LogLatency:      true,
		LogReferer:      true,
		LogUserAgent:    true,
		LogRequestID:    true,
		LogValuesFunc: func(c echo.Context, v echomiddelware.RequestLoggerValues) error {
			if v.Status >= http.StatusInternalServerError && logLevel < zerolog.ErrorLevel {
				logLevel = zerolog.ErrorLevel
			} else if v.Status >= http.StatusBadRequest && logLevel < zerolog.WarnLevel {
				logLevel = zerolog.WarnLevel
			}

			logger.WithLevel(logLevel).
				Str("RequestID", v.RequestID).
				Str("Method", v.Method).
				Str("URI", v.URI).
				Str("RemoteAddr", v.RemoteIP).
				Int("StatusCode", v.Status).
				Int64("Size", v.ResponseSize).
				Dur("Duration", v.Latency).
				Str("Referer", v.Referer).
				Str("UserAgent", v.UserAgent).
				Str("ClientID", c.Response().Header().Get(apimodels.HTTPHeaderClientID)).
				Str("JobID", c.Response().Header().Get(apimodels.HTTPHeaderJobID)).
				Send()
			return nil
		},
	})
}
