package requester

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
)

// debug godoc
//
//	@ID			pkg/requester/publicapi/debug
//	@Summary	Returns debug information on what the current node is doing.
//	@Tags		Health
//	@Produce	json
//	@Success	200	{object}	string
//	@Failure	500	{object}	string
//	@Router		/api/v1/requester/debug [get]
func (s *Endpoint) debug(res http.ResponseWriter, req *http.Request) {
	debugInfoMap := make(map[string]interface{})
	for _, provider := range s.debugInfoProviders {
		debugInfo, err := provider.GetDebugInfo(req.Context())
		if err != nil {
			log.Ctx(req.Context()).Error().Msgf("could not get debug info from some providers: %s", err)
			continue
		}
		debugInfoMap[debugInfo.Component] = debugInfo.Info
	}

	render.JSON(res, req, debugInfoMap)
}
