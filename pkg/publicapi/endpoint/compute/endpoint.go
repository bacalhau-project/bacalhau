package compute

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type EndpointParams struct {
	Router             chi.Router
	Bidder             compute.Bidder
	Store              store.ExecutionStore
	DebugInfoProviders []model.DebugInfoProvider
}

type Endpoint struct {
	router             chi.Router
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

	e.router.Route("/api/v1/compute", func(r chi.Router) {
		r.Use(render.SetContentType(render.ContentTypeJSON))
		r.Post("/debug", e.debug)
		r.Post("/approve", e.approve)
	})
	return e
}
