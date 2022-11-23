package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

// Returns debug information on what the current node is doing.
func (apiServer *APIServer) debug(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/debug")
	defer span.End()

	debugInfoMap := make(map[string]interface{})
	for _, provider := range apiServer.DebugInfoProviders {
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
