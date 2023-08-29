package compute

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/labstack/echo/v4"
)

type EndpointParams struct {
	Router             *echo.Echo
	Bidder             compute.Bidder
	Store              store.ExecutionStore
	DebugInfoProviders []model.DebugInfoProvider
}

type Endpoint struct {
	router             *echo.Echo
	bidder             compute.Bidder
	store              store.ExecutionStore
	debugInfoProviders []model.DebugInfoProvider
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:             params.Router,
		bidder:             params.Bidder,
		store:              params.Store,
		debugInfoProviders: params.DebugInfoProviders,
	}

	g := e.router.Group("/api/v1/compute")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.POST("/debug", e.debug)
	g.POST("/approve", e.approve)
	return e
}
