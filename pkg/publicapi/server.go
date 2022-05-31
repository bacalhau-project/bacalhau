package publicapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// APIServer configures a node's public REST API.
type APIServer struct {
	Node *requestor_node.RequesterNode
	Host string
	Port int

	ctx *system.CancelContext
}

// NewServer returns a new API server for a requester node.
func NewServer(
	node *requestor_node.RequesterNode,
	host string,
	port int,
) *APIServer {
	return &APIServer{
		Host: host,
		Port: port,
		Node: node,
	}
}

// Start listens for and serves HTTP requests against the API server.
func (apiServer *APIServer) Start(ctx *system.CancelContext) error {
	hostID, err := apiServer.Node.Transport.HostId()
	if err != nil {
		log.Error().Msgf("Error fetching node's host ID: %s", err)
		return err
	}

	httpServer := http.Server{
		Addr: fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
	}

	ctx.AddShutdownHandler(func() {
		err := httpServer.Close()
		if err != nil {
			log.Error().Msgf(
				"Error shutting down API server for host %s: %s", hostID, err)
		}
	})

	log.Info().Msgf(
		"API server listening for host %s on %s...", hostID, httpServer.Addr)
	return httpServer.ListenAndServe()
}

type listRequest struct{}
type listResponse struct{}

func (apiServer *APIServer) list(res http.ResponseWriter, req *http.Request) {
	var listReq listRequest
	if err := json.NewDecoder(req.Body).Decode(&listReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	list, err := apiServer.Node.Transport.List()
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

    json.NewEncoder(res).Encode(list)
}

type submitRequest struct{}
type submitResponse struct{}

func (apiServer *APIServer) submit(res http.ResponseWriter, req *http.Request) {
	var submitReq submitRequest
	if err := json.NewDecoder(req.Body).Decode(&submitReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
}
