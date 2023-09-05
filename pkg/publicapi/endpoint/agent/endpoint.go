package agent

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/labstack/echo/v4"
)

type EndpointParams struct {
	Router           *echo.Echo
	NodeInfoProvider models.NodeInfoProvider
}

type Endpoint struct {
	router           *echo.Echo
	nodeInfoProvider models.NodeInfoProvider
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:           params.Router,
		nodeInfoProvider: params.NodeInfoProvider,
	}

	// JSON group
	g := e.router.Group("/api/v1/agent")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.GET("/alive", e.alive)
	g.GET("/version", e.version)
	g.GET("/node", e.node)
	return e
}

// alive godoc
//
//	@ID			agent/alive
//	@Tags		Ops
//	@Produce	text/plain
//	@Success	200	{string}	string	"OK"
//	@Router		/api/v1/agent/alive [get]
func (e *Endpoint) alive(c echo.Context) error {
	return c.JSON(http.StatusOK, &apimodels.IsAliveResponse{
		Status: "OK",
	})
}

// version godoc
//
//	@ID				agent/version
//	@Summary		Returns the build version running on the server.
//	@Description	See https://github.com/bacalhau-project/bacalhau/releases for a complete list of `gitversion` tags.
//	@Tags			Ops
//	@Produce		json
//	@Success		200				{object}	apimodels.GetVersionResponse
//	@Failure		500				{object}	json
//	@Router			/api/v1/agent/version [get]
func (e *Endpoint) version(c echo.Context) error {
	return c.JSON(http.StatusOK, apimodels.GetVersionResponse{
		BuildVersionInfo: version.Get(),
	})
}

// node godoc
//
//	@ID			agent/node
//	@Summary	Returns the info of the node.
//	@Tags		Ops
//	@Produce	json
//	@Success	200	{object}	models.NodeInfo
//	@Failure	500	{object}	json
//	@Router		/api/v1/agent/node [get]
func (e *Endpoint) node(c echo.Context) error {
	nodeInfo := e.nodeInfoProvider.GetNodeInfo(c.Request().Context())
	return c.JSON(http.StatusOK, apimodels.GetAgentNodeResponse{
		NodeInfo: &nodeInfo,
	})
}
