package agent

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type EndpointParams struct {
	Router             *echo.Echo
	NodeInfoProvider   models.NodeInfoProvider
	DebugInfoProviders []model.DebugInfoProvider
}

type Endpoint struct {
	router             *echo.Echo
	nodeInfoProvider   models.NodeInfoProvider
	debugInfoProviders []model.DebugInfoProvider
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:             params.Router,
		nodeInfoProvider:   params.NodeInfoProvider,
		debugInfoProviders: params.DebugInfoProviders,
	}

	// JSON group
	g := e.router.Group("/api/v1/agent")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.GET("/alive", e.alive)
	g.GET("/version", e.version)
	g.GET("/node", e.node)
	g.GET("/debug", e.debug)
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
//	@Success		200	{object}	apimodels.GetVersionResponse
//	@Failure		500	{object}	string
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
//	@Failure	500	{object}	string
//	@Router		/api/v1/agent/node [get]
func (e *Endpoint) node(c echo.Context) error {
	nodeInfo := e.nodeInfoProvider.GetNodeInfo(c.Request().Context())
	return c.JSON(http.StatusOK, apimodels.GetAgentNodeResponse{
		NodeInfo: &nodeInfo,
	})
}

// debug godoc
//
//	@ID			agent/debug
//	@Summary	Returns debug information on what the current node is doing.
//	@Tags		Ops
//	@Produce	json
//	@Success	200	{object}	model.DebugInfo
//	@Failure	500	{object}	string
//	@Router		/api/v1/agent/debug [get]
func (e *Endpoint) debug(c echo.Context) error {
	debugInfoMap := make(map[string]interface{})
	for _, provider := range e.debugInfoProviders {
		debugInfo, err := provider.GetDebugInfo(c.Request().Context())
		if err != nil {
			log.Ctx(c.Request().Context()).Error().Msgf("could not get debug info from some providers: %s", err)
			continue
		}
		debugInfoMap[debugInfo.Component] = debugInfo.Info
	}
	return c.JSON(http.StatusOK, debugInfoMap)
}
