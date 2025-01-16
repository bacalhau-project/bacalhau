package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/license"
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

func NewEndpoint(params EndpointParams) *Endpoint {
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
	g.GET("/license", e.license)
	return e
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
	cfg, err := e.bacalhauConfig.Copy()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("could not copy bacalhau config: %s", err))
	}
	if cfg.Compute.Auth.Token != "" {
		cfg.Compute.Auth.Token = "<redacted>"
	}
	if cfg.Orchestrator.Auth.Token != "" {
		cfg.Orchestrator.Auth.Token = "<redacted>"
	}
	return c.JSON(http.StatusOK, apimodels.GetAgentConfigResponse{
		Config: cfg,
	})
}

// license godoc
//
//	@ID			agent/license
//	@Summary	Returns the details of the current configured orchestrator license.
//	@Tags		Ops
//	@Produce	json
//	@Success	200	{object}	license.LicenseClaims
//	@Failure	404	{object}	string	"Node license not configured"
//	@Failure	500	{object}	string
//	@Router		/api/v1/agent/license [get]
func (e *Endpoint) license(c echo.Context) error {
	// Get license path from config
	licensePath := e.bacalhauConfig.Orchestrator.License.LocalPath
	if licensePath == "" {
		return echo.NewHTTPError(http.StatusNotFound, "Node license not configured")
	}

	// Read license file
	licenseData, err := os.ReadFile(licensePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to read license file: %s", err))
	}

	// Parse license JSON
	var licenseFile struct {
		License string `json:"license"`
	}
	if err := json.Unmarshal(licenseData, &licenseFile); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to parse license file: %s", err))
	}

	// Create validator and validate license
	validator, err := license.NewOfflineLicenseValidator()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create license validator: %s", err))
	}

	claims, err := validator.ValidateToken(licenseFile.License)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to validate license: %s", err))
	}

	return c.JSON(http.StatusOK, apimodels.GetAgentLicenseResponse{
		LicenseClaims: claims,
	})
}
