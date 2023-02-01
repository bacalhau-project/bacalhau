package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/system"
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
//	@Router		/requester/debug [get]
func (s *RequesterAPIServer) debug(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/debug")
	defer span.End()

	debugInfoMap := make(map[string]interface{})
	for _, provider := range s.debugInfoProviders {
		debugInfo, err := provider.GetDebugInfo()
		if err != nil {
			log.Ctx(ctx).Error().Msgf("could not get debug info from some providers: %s", err)
			continue
		}
		debugInfoMap[debugInfo.Component] = debugInfo.Info
	}

	res.WriteHeader(http.StatusOK)
	err := json.NewEncoder(res).Encode(debugInfoMap)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
