package middleware

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

func RequestLogger(onlyErrorStatuses bool) echo.MiddlewareFunc {
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
			if onlyErrorStatuses && v.Status < http.StatusBadRequest {
				return nil
			}
			log.Ctx(c.Request().Context()).Info().
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
