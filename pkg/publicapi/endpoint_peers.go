package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

// peers godoc
// @ID                   apiServer/peers
// @Summary              Returns the peers connected to the host via the libp2pHost layer.
// @Description.markdown endpoints_peers
// @Tags                 Misc
// @Produce              json
// @Success              200 {object} map[string][]string{}
// @Failure              500 {object} string
// @Router               /peers [get]
func (apiServer *APIServer) peers(res http.ResponseWriter, req *http.Request) {
	_, span := system.GetSpanFromRequest(req, "apiServer/peers")
	defer span.End()
	res.WriteHeader(http.StatusOK)
	err := json.NewEncoder(res).Encode(apiServer.host.Peerstore().Peers())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
