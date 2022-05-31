package publicapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
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

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *APIServer) GetURI() string {
	return fmt.Sprintf("http://%s:%d", apiServer.Host, apiServer.Port)
}

// Start listens for and serves HTTP requests against the API server.
func (apiServer *APIServer) Start(ctx *system.CancelContext) error {
	hostID, err := apiServer.Node.Transport.HostId()
	if err != nil {
		log.Error().Msgf("Error fetching node's host ID: %s", err)
		return err
	}

	sm := http.NewServeMux()
	sm.Handle("/list", http.HandlerFunc(apiServer.list))
	sm.Handle("/submit", http.HandlerFunc(apiServer.submit))
	sm.Handle("/health", http.HandlerFunc(apiServer.health))

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
		Handler: sm,
	}

	ctx.AddShutdownHandler(func() {
		err := srv.Close()
		if err != nil {
			log.Error().Msgf(
				"Error shutting down API server for host %s: %s", hostID, err)
		}
	})

	log.Info().Msgf(
		"API server listening for host %s on %s...", hostID, srv.Addr)
	return srv.ListenAndServe()
}

func (apiServer *APIServer) health(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}

type listRequest struct{}

type listResponse struct {
	Jobs map[string]*types.Job `json:"jobs"`
}

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

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(listResponse{
		Jobs: list.Jobs,
	})
}

type submitRequest struct {
	Spec *types.JobSpec `json:"spec"`
	Deal *types.JobDeal `json:"deal"`
}

type submitResponse struct {
	Job *types.Job `json:"job"`
}

func (apiServer *APIServer) submit(res http.ResponseWriter, req *http.Request) {
	var submitReq submitRequest
	if err := json.NewDecoder(req.Body).Decode(&submitReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := job.VerifyJob(submitReq.Spec, submitReq.Deal); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	job, err := apiServer.Node.Transport.SubmitJob(submitReq.Spec, submitReq.Deal)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(submitResponse{
		Job: job,
	})
}
