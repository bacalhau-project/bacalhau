package compute

import (
	"github.com/labstack/echo/v4"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
)

type EndpointParams struct {
	Router             *echo.Echo
	DebugInfoProviders []models.DebugInfoProvider
}

type Endpoint struct {
	router             *echo.Echo
	debugInfoProviders []models.DebugInfoProvider
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:             params.Router,
		debugInfoProviders: params.DebugInfoProviders,
	}

	g := e.router.Group("/api/v1/compute")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.POST("/debug", e.debug)
	return e
}
