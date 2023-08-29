package compute

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// debug godoc
//
//	@ID			apiServer/debug
//	@Summary	Returns debug information on what the current node is doing.
//	@Tags		Health
//	@Produce	json
//	@Success	200	{object}	string
//	@Failure	500	{object}	string
//	@Router		/api/v1/compute/debug [get]
func (s *Endpoint) debug(c echo.Context) error {
	debugInfoMap := make(map[string]interface{})
	for _, provider := range s.debugInfoProviders {
		debugInfo, err := provider.GetDebugInfo(c.Request().Context())
		if err != nil {
			log.Ctx(c.Request().Context()).Error().Msgf("could not get debug info from some providers: %s", err)
			continue
		}
		debugInfoMap[debugInfo.Component] = debugInfo.Info
	}
	return c.JSON(http.StatusOK, debugInfoMap)
}
