package middleware

import (
	"github.com/labstack/echo/v4"
)

// SetContentType returns a middleware which sets the response content type.
func SetCrossOrigin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
			return next(c)
		}
	}
}
