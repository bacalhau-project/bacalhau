package middleware

import (
	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/labstack/echo/v4"
)

const AuthorizationComponent = "Authorizer"

// Authorize only allows the HTTP request to continue if the passed authorizer
// permits the request.
func Authorize(authorizer authz.Authorizer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if result, err := authorizer.Authorize(c.Request()); err != nil {
				return bacerrors.New("unexpected authorization error").
					WithCode(bacerrors.InternalError).
					WithComponent(AuthorizationComponent).
					WithHint("Please check orchestrator logs for more details")
			} else if !result.Approved && result.TokenValid {
				return bacerrors.New("Request Forbidden").
					WithCode(bacerrors.Forbidden).
					WithComponent(AuthorizationComponent).
					WithDetail("reason", result.Reason).
					WithHint("Check if user have access to this resource. Event has been recorded.")
			} else if !result.Approved && !result.TokenValid {
				return bacerrors.New("Request Unauthorized").
					WithCode(bacerrors.UnauthorizedError).
					WithComponent(AuthorizationComponent).
					WithDetail("reason", result.Reason).
					WithHint("Unauthorized principal. Event has been recorded.")
			} else {
				return next(c)
			}
		}
	}
}
