package auth

import (
	"context"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
)

type Endpoint struct {
	router   *echo.Echo
	provider provider.Provider[authn.Authenticator]
}

func BindEndpoint(ctx context.Context, router *echo.Echo, provider authn.Provider) *Endpoint {
	e := &Endpoint{
		router:   router,
		provider: provider,
	}

	g := e.router.Group("/api/v1/auth")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.GET("", e.list)

	// Each key is the operator-specified name of a configured authentication
	// method. See the documentation on authn.Provider for more.
	for _, name := range provider.Keys(ctx) {
		authenticator := lo.Must(provider.Get(ctx, name))
		adaptAuthenticator(authenticator, g.Group("/"+name))
	}

	return e
}

func adaptAuthenticator(method authn.Authenticator, route *echo.Group) {
	route.GET("", func(c echo.Context) error {
		return c.JSON(http.StatusOK, method.Requirement())
	})

	route.POST("", func(c echo.Context) error {
		var req apimodels.AuthnRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if req.Headers == nil {
			req.Headers = make(map[string]string)
		}
		req.Headers["Authorization"] = c.Request().Header.Get("Authorization")

		if err := c.Validate(&req); err != nil {
			return err
		}

		authentication, err := method.Authenticate(c.Request().Context(), req.MethodData)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		result := apimodels.AuthnResponse{Authentication: authentication}
		if !authentication.Success {
			return c.JSON(http.StatusUnauthorized, result)
		} else {
			return c.JSON(http.StatusOK, result)
		}
	})
}

func (e *Endpoint) list(c echo.Context) error {
	methods := lo.SliceToMap(
		e.provider.Keys(c.Request().Context()),
		func(item string) (string, authn.Requirement) {
			provider := lo.Must(e.provider.Get(c.Request().Context(), item))
			return item, provider.Requirement()
		},
	)

	return c.JSON(http.StatusOK, apimodels.ListAuthnMethodsResponse{Methods: methods})
}
