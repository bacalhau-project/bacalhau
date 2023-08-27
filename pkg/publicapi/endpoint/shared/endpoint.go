package shared

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/docs"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	httpSwagger "github.com/swaggo/http-swagger"
)

type EndpointParams struct {
	Router           chi.Router
	NodeID           string
	PeerStore        peerstore.Peerstore
	NodeInfoProvider models.NodeInfoProvider
}

type Endpoint struct {
	router           chi.Router
	nodeID           string
	peerStore        peerstore.Peerstore
	nodeInfoProvider models.NodeInfoProvider
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:           params.Router,
		nodeID:           params.NodeID,
		peerStore:        params.PeerStore,
		nodeInfoProvider: params.NodeInfoProvider,
	}

	e.router.Route("/api/v1", func(r chi.Router) {
		// group for JSON endpoints
		r.Group(func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypeJSON))
			r.Get("/peers", e.peers)
			r.Get("/node_info", e.nodeInfo)
			r.Post("/version", e.version)
			r.Get("/healthz", e.healthz)
		})
		// group for plaintext endpoints
		r.Group(func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypePlainText))
			r.Get("/id", e.id)
			r.Get("/livez", e.livez)
		})
	})

	// swagger UI
	// dynamically write the git tag to the Swagger docs
	docs.SwaggerInfo.Version = version.Get().GitVersion
	// swagger docs at root
	e.router.Mount("/swagger/", httpSwagger.WrapHandler)

	return e
}

// id godoc
//
//	@ID			id
//	@Summary	Returns the id of the host node.
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string
//	@Failure	500	{object}	string
//	@Router		/api/v1/id [get]
func (e *Endpoint) id(w http.ResponseWriter, r *http.Request) {
	render.PlainText(w, r, e.nodeID)
}

// peers godoc
//
//	@ID						peers
//	@Summary				Returns the peers connected to the host via the transport layer.
//	@Description.markdown	endpoints_peers
//	@Tags					Utils
//	@Produce				json
//	@Success				200	{object}	[]peer.AddrInfo
//	@Failure				500	{object}	string
//	@Router					/api/v1/peers [get]
func (e *Endpoint) peers(w http.ResponseWriter, r *http.Request) {
	var peerInfos []peer.AddrInfo
	for _, p := range e.peerStore.Peers() {
		peerInfos = append(peerInfos, e.peerStore.PeerInfo(p))
	}
	render.JSON(w, r, peerInfos)
}

// nodeInfo godoc
//
//	@ID			nodeInfo
//	@Summary	Returns the info of the node.
//	@Tags		Utils
//	@Produce	json
//	@Success	200	{object}	model.NodeInfo
//	@Failure	500	{object}	string
//	@Router		/api/v1/node_info [get]
func (e *Endpoint) nodeInfo(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, e.nodeInfoProvider.GetNodeInfo(r.Context()))
}

// version godoc
//
//	@ID				apiServer/version
//	@Summary		Returns the build version running on the server.
//	@Description	See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.
//	@Tags			Misc
//	@Accept			json
//	@Produce		json
//	@Param			VersionRequest	body		VersionRequest	true	"Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field."
//	@Success		200				{object}	VersionResponse
//	@Failure		400				{object}	string
//	@Failure		500				{object}	string
//	@Router			/api/v1/version [post]
//
//nolint:lll
func (e *Endpoint) version(w http.ResponseWriter, r *http.Request) {
	versionReq := &apimodels.VersionRequest{}
	if err := render.DecodeJSON(r.Body, versionReq); err != nil {
		publicapi.HTTPError(r.Context(), w, err, http.StatusBadRequest)
		return
	}

	render.JSON(w, r, apimodels.VersionResponse{
		VersionInfo: version.Get(),
	})
}

// healthz godoc
//
//	@ID			healthz
//	@Tags		Utils
//	@Produce	json
//	@Success	200	{object}	types.HealthInfo
//	@Router		/api/v1/healthz [get]
func (e *Endpoint) healthz(w http.ResponseWriter, r *http.Request) {
	// TODO: A list of health information. Should require authing (of some kind)
	// Ideas:
	// CPU usage
	render.JSON(w, r, GenerateHealthData())
}

// livez godoc
//
//	@ID			livez
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string	"TODO"
//	@Router		/api/v1/livez [get]
func (e *Endpoint) livez(w http.ResponseWriter, r *http.Request) {
	// Extremely simple liveness check (should be fine to be public / no-auth)
	render.PlainText(w, r, "OK")
}
