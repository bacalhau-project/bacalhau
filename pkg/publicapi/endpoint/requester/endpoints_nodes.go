package requester

import (
	"encoding/json"
	"net/http"
)

// nodes godoc
//
//	@ID			pkg/requester/publicapi/nodes
//	@Summary	Displays the nodes that this requester knows about
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	[]models.NodeInfo
//	@Failure	500	{object}	string
//	@Router		/api/v1/requester/nodes [get]
func (s *Endpoint) nodes(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	nodes, err := s.nodeDiscoverer.ListNodes(ctx)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(res).Encode(nodes)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
