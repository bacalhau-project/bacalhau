package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ServerHeader middleware adds HTTP Headers `headers` to response
func ServerHeader(headers map[string]string, skippers ...middleware.Skipper) echo.MiddlewareFunc {
	chainedSkipper := ChainedSkipper(skippers...)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !chainedSkipper(c) {
				for key, header := range headers {
					c.Response().Header().Set(key, header)
				}
			}
			return next(c)
		}
	}
}
