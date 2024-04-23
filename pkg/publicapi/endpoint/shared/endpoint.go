package shared

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type EndpointParams struct {
	Router            *echo.Echo
	NodeID            string
	NodeStateProvider models.NodeStateProvider
}

type Endpoint struct {
	router            *echo.Echo
	nodeID            types.NodeID
	nodeStateProvider models.NodeStateProvider
}

type SharedEndpoingParams struct {
	fx.In

	NodeID       types.NodeID
	Router       *echo.Echo
	NodeProvider *routing.NodeStateProvider
}

func InitSharedEndpoint(p SharedEndpoingParams) {
	shared := &Endpoint{
		nodeID:            p.NodeID,
		nodeStateProvider: p.NodeProvider,
	}

	// JSON group
	g := p.Router.Group("/api/v1")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.GET("/node_info", shared.nodeInfo)
	g.POST("/version", shared.version)
	g.GET("/healthz", shared.healthz)

	// Plaintext group
	pt := p.Router.Group("/api/v1")
	pt.Use(middleware.SetContentType(echo.MIMETextPlain))
	pt.GET("/id", shared.id)
	pt.GET("/livez", shared.livez)

	// Home group
	// TODO: Could we use this to redirect to latest API?
	h := p.Router.Group("/")
	h.Use(middleware.SetContentType(echo.MIMETextPlain))
	h.GET("", shared.home)
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:            params.Router,
		nodeID:            types.NodeID(params.NodeID),
		nodeStateProvider: params.NodeStateProvider,
	}

	// JSON group
	g := e.router.Group("/api/v1")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.GET("/node_info", e.nodeInfo)
	g.POST("/version", e.version)
	g.GET("/healthz", e.healthz)

	// Plaintext group
	pt := e.router.Group("/api/v1")
	pt.Use(middleware.SetContentType(echo.MIMETextPlain))
	pt.GET("/id", e.id)
	pt.GET("/livez", e.livez)

	// Home group
	// TODO: Could we use this to redirect to latest API?
	h := e.router.Group("/")
	h.Use(middleware.SetContentType(echo.MIMETextPlain))
	h.GET("", e.home)

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
func (e *Endpoint) id(c echo.Context) error {
	return c.String(http.StatusOK, string(e.nodeID))
}

// nodeInfo godoc
//
//	@ID			nodeInfo
//	@Summary	Returns the info of the node.
//	@Tags		Utils
//	@Produce	json
//	@Success	200	{object}	models.NodeInfo
//	@Failure	500	{object}	string
//	@Router		/api/v1/node_info [get]
func (e *Endpoint) nodeInfo(c echo.Context) error {
	return c.JSON(http.StatusOK, e.nodeStateProvider.GetNodeState(c.Request().Context()))
}

// version godoc
//
//	@ID				apiServer/version
//	@Summary		Returns the build version running on the server.
//	@Description	See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.
//	@Tags			Misc
//	@Accept			json
//	@Produce		json
//	@Param			VersionRequest	body		legacymodels.VersionRequest	true	"Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field."
//	@Success		200				{object}	legacymodels.VersionResponse
//	@Failure		400				{object}	string
//	@Failure		500				{object}	string
//	@Router			/api/v1/version [post]
//
//nolint:lll
func (e *Endpoint) version(c echo.Context) error {
	var versionReq legacymodels.VersionRequest
	if err := c.Bind(&versionReq); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, legacymodels.VersionResponse{
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
func (e *Endpoint) healthz(c echo.Context) error {
	// TODO: A list of health information. Should require authing (of some kind)
	// Ideas:
	// CPU usage
	return c.JSON(http.StatusOK, GenerateHealthData())
}

// livez godoc
//
//	@ID			livez
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string	"TODO"
//	@Router		/api/v1/livez [get]
func (e *Endpoint) livez(c echo.Context) error {
	// Extremely simple liveness check (should be fine to be public / no-auth)
	return c.String(http.StatusOK, "OK")
}

// home godoc
//
//	@ID			home
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string	""
//	@Router		/ [get]
func (e *Endpoint) home(c echo.Context) error {
	return c.JSON(http.StatusOK, version.Get())
}
