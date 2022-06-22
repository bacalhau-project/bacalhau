package publicapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// APIServer configures a node's public REST API.
type APIServer struct {
	Node *requestor_node.RequesterNode
	Host string
	Port int
}

// NewServer returns a new API server for a requester node.
func NewServer(
	node *requestor_node.RequesterNode,
	host string,
	port int,
) *APIServer {
	return &APIServer{
		Node: node,
		Host: host,
		Port: port,
	}
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *APIServer) GetURI() string {
	return fmt.Sprintf("http://%s:%d", apiServer.Host, apiServer.Port)
}

// ListenAndServe listens for and serves HTTP requests against the API server.
func (apiServer *APIServer) ListenAndServe(ctx context.Context,
	cm *system.CleanupManager) error {

	hostID, err := apiServer.Node.Transport.HostID(ctx)
	if err != nil {
		log.Error().Msgf("Error fetching node's host ID: %s", err)
		return err
	}

	sm := http.NewServeMux()
	sm.Handle("/list", instrument("list", apiServer.list))
	sm.Handle("/submit", instrument("submit", apiServer.submit))
	sm.Handle("/healthz", instrument("healthz", apiServer.healthz))
	sm.Handle("/logz", instrument("logz", apiServer.logz))
	sm.Handle("/varz", instrument("varz", apiServer.varz))
	sm.Handle("/livez", instrument("livez", apiServer.livez))
	sm.Handle("/readyz", instrument("readyz", apiServer.readyz))

	srv := http.Server{
		Handler: sm,
		Addr:    fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
	}

	log.Debug().Msgf(
		"API server listening for host %s on %s...", hostID, srv.Addr)

	// Cleanup resources when system is done:
	cm.RegisterCallback(func() error {
		return srv.Shutdown(ctx)
	})

	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		log.Debug().Msgf(
			"API server closed for host %s on %s.", hostID, srv.Addr)
		return nil // expected error if the server is shut down
	}

	return err
}

type listRequest struct{}

type listResponse struct {
	Jobs map[string]*executor.Job `json:"jobs"`
}

func (apiServer *APIServer) list(res http.ResponseWriter, req *http.Request) {
	var listReq listRequest
	if err := json.NewDecoder(req.Body).Decode(&listReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	list, err := apiServer.Node.Transport.List(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(listResponse{
		Jobs: list.Jobs,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

type submitRequest struct {
	Spec *executor.JobSpec `json:"spec"`
	Deal *executor.JobDeal `json:"deal"`
}

type submitResponse struct {
	Job *executor.Job `json:"job"`
}

func (apiServer *APIServer) submit(res http.ResponseWriter, req *http.Request) {
	var submitReq submitRequest
	if err := json.NewDecoder(req.Body).Decode(&submitReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := job.VerifyJob(submitReq.Spec, submitReq.Deal); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	job, err := apiServer.Node.Transport.SubmitJob(req.Context(),
		submitReq.Spec, submitReq.Deal)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(submitResponse{
		Job: job,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func instrument(name string, fn http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(fn, fmt.Sprintf("publicapi/%s", name))
}
