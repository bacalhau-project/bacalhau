package shared

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type EndpointParams struct {
	Router           *echo.Echo
	NodeID           string
	NodeInfoProvider models.NodeInfoProvider
}

type Endpoint struct {
	router           *echo.Echo
	nodeID           string
	nodeInfoProvider models.NodeInfoProvider
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:           params.Router,
		nodeID:           params.NodeID,
		nodeInfoProvider: params.NodeInfoProvider,
	}

	// JSON group
	g := e.router.Group("/api/v1")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.GET("/node_info", e.nodeInfo)
	g.POST("/version", e.version)

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
	return c.String(http.StatusOK, e.nodeID)
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
	return c.JSON(http.StatusOK, e.nodeInfoProvider.GetNodeInfo(c.Request().Context()))
}

type VersionRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
}

type VersionResponse struct {
	VersionInfo *models.BuildVersionInfo `json:"build_version_info"`
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
func (e *Endpoint) version(c echo.Context) error {
	var versionReq VersionRequest
	if err := c.Bind(&versionReq); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, VersionResponse{
		VersionInfo: version.Get(),
	})
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
