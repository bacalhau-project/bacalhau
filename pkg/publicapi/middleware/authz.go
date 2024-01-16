package middleware

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/labstack/echo/v4"
)

// Authorize only allows the HTTP request to continue if the passed authorizer
// permits the request.
func Authorize(authorizer authz.Authorizer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if result, err := authorizer.Authorize(c.Request()); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			} else if !result.Approved {
				return echo.NewHTTPError(http.StatusForbidden, result.Reason)
			} else {
				return next(c)
			}
		}
	}
}
