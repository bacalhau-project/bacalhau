package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

// id godoc
// @ID      apiServer/id
// @Summary Returns the id of the host node.
// @Tags    Misc
// @Produce text/plain
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router  /id [get]
func (apiServer *APIServer) id(res http.ResponseWriter, req *http.Request) {
	_, span := system.GetSpanFromRequest(req, "apiServer/id")
	defer span.End()

	res.WriteHeader(http.StatusOK)
	err := json.NewEncoder(res).Encode(apiServer.libp2pHost.ID().String())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
