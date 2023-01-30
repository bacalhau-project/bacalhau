package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

// nodeInfo godoc
//
// @ID      nodeInfo
// @Summary Returns the info of the node.
// @Tags    Utils
// @Produce json
// @Success 200 {object} model.NodeInfo
// @Failure 500 {object} string
// @Router  /node_info [get]
func (apiServer *APIServer) nodeInfo(res http.ResponseWriter, req *http.Request) {
	_, span := system.GetSpanFromRequest(req, "apiServer/node_info")
	defer span.End()
	res.WriteHeader(http.StatusOK)
	err := json.NewEncoder(res).Encode(apiServer.nodeInfoProvider.GetNodeInfo(req.Context()))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
