package publicapi

import (
	"encoding/json"
	"net/http"
)

// id godoc
//
//	@ID			id
//	@Summary	Returns the id of the host node.
//	@Tags		Utils
//	@Produce	text/plain
//	@Success	200	{object}	string
//	@Failure	500	{object}	string
//	@Router		/id [get]
func (apiServer *APIServer) id(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
	err := json.NewEncoder(res).Encode(apiServer.host.ID().String())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
