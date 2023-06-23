package publicapi

import (
	"encoding/json"
	"net/http"
)

// nodes godoc
//
//	@ID						pkg/requester/publicapi/nodes
//	@Summary				Displays the nodes that this requester knows about
//	@Description.markdown	endpoints_nodes
//	@Accept					json
//	@Produce				json
//	@Success				200				{object}	[]model.NodeInfo
//	@Failure				500				{object}	string
//	@Router					/requester/nodes [get]
func (s *RequesterAPIServer) nodes(res http.ResponseWriter, req *http.Request) {
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
