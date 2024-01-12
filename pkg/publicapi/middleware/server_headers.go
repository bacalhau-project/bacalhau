package middleware

import (
	"github.com/labstack/echo/v4"
)

// ServerHeader middleware adds HTTP Headers `headers` to response
func ServerHeader(headers map[string]string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for key, header := range headers {
				c.Response().Header().Set(key, header)
			}
			return next(c)
		}
	}
}
