package agent

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type EndpointParams struct {
	Router             *echo.Echo
	NodeInfoProvider   models.NodeInfoProvider
	DebugInfoProviders []models.DebugInfoProvider
	BacalhauConfig     types.Bacalhau
}

type Endpoint struct {
	router             *echo.Echo
	nodeInfoProvider   models.NodeInfoProvider
	debugInfoProviders []models.DebugInfoProvider
	bacalhauConfig     types.Bacalhau
}

func NewEndpoint(params EndpointParams) (*Endpoint, error) {
	e := &Endpoint{
		router:             params.Router,
		nodeInfoProvider:   params.NodeInfoProvider,
		debugInfoProviders: params.DebugInfoProviders,
		bacalhauConfig:     params.BacalhauConfig,
	}

	// JSON group
	g := e.router.Group("/api/v1/agent")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.GET("/alive", e.alive)
	g.GET("/version", e.version)
	g.GET("/node", e.node)
	g.GET("/debug", e.debug)
	g.GET("/config", e.config)
	g.GET("/authconfig", e.nodeAuthConfig)

	return e, nil
}

// alive godoc
//
//	@ID			agent/alive
//	@Tags		Ops
//	@Produce	text/plain
//	@Success	200	{object}	apimodels.IsAliveResponse
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
//	@Success	200	{object}	models.DebugInfo
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

// config godoc
//
//	@ID			agent/config
//	@Summary	Returns the current configuration of the node.
//	@Tags		Ops
//	@Produce	json
//	@Success	200	{object}	types.Bacalhau
//	@Failure	500	{object}	string
//	@Router		/api/v1/agent/config [get]
func (e *Endpoint) config(c echo.Context) error {
	clonedRedactedConfig, err := redactConfigSensitiveInfo(e.bacalhauConfig)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("could not copy bacalhau config: %s", err))
	}
	return c.JSON(http.StatusOK, apimodels.GetAgentConfigResponse{
		Config: clonedRedactedConfig,
	})
}

// nodeAuthConfig godoc
//
//	@ID			agent/authconfig
//	@Summary	Returns the OAuth2 configuration of the node.
//	@Tags		Ops
//	@Produce	json
//	@Success	200	{object}	apimodels.GetAgentNodeAuthConfigResponse
//	@Failure	500	{object}	string
//	@Router		/api/v1/agent/authconfig [get]
func (e *Endpoint) nodeAuthConfig(c echo.Context) error {
	// No need to redact Oauth2 config since these are made to be public
	return c.JSON(http.StatusOK, apimodels.GetAgentNodeAuthConfigResponse{
		Version: "1.0.0",
		Config:  e.bacalhauConfig.API.Auth.Oauth2,
	})
}
