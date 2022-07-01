package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/requestornode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// APIServer configures a node's public REST API.
type APIServer struct {
	Node *requestornode.RequesterNode
	Host string
	Port int
}

// NewServer returns a new API server for a requester node.
func NewServer(
	node *requestornode.RequesterNode,
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
func (apiServer *APIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
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

type listRequest struct {
	ClientID string `json:"client_id"`
}

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
		return
	}
}

type submitData struct {
	// The job specification:
	Spec *executor.JobSpec `json:"spec"`

	// The deal the client has made with the network, at minimum this should
	// contain the client's ID for verifying the message authenticity:
	Deal *executor.JobDeal `json:"deal"`
}

type submitRequest struct {
	// The data needed to submit and run a job on the network:
	Data submitData `json:"data"`

	// A base64-encoded signature of the data, signed by the client:
	ClientSignature string `json:"signature"`

	// The base64-encoded public key of the client:
	ClientPublicKey string `json:"client_public_key"`
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

	if err := verifySubmitRequest(&submitReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := job.VerifyJob(submitReq.Data.Spec, submitReq.Data.Deal); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	j, err := apiServer.Node.Transport.SubmitJob(req.Context(),
		submitReq.Data.Spec, submitReq.Data.Deal)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(submitResponse{
		Job: j,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func verifySubmitRequest(req *submitRequest) error {
	if req.Data.Spec == nil {
		return errors.New("job spec is required")
	}
	if req.Data.Deal == nil {
		return errors.New("job deal is required")
	}
	if req.Data.Deal.ClientID == "" {
		return errors.New("job deal must contain a client ID")
	}
	if req.ClientSignature == "" {
		return errors.New("client's signature is required")
	}
	if req.ClientPublicKey == "" {
		return errors.New("client's public key is required")
	}

	// Check that the client's public key matches the client ID:
	ok, err := system.PublicKeyMatchesID(req.ClientPublicKey, req.Data.Deal.ClientID)
	if err != nil {
		return fmt.Errorf("error verifying client ID: %w", err)
	}
	if !ok {
		return errors.New("client's public key does not match client ID")
	}

	// Check that the signature is valid:
	jsonData, err := json.Marshal(req.Data)
	if err != nil {
		return fmt.Errorf("error marshaling job data: %w", err)
	}

	ok, err = system.Verify(jsonData, req.ClientSignature, req.ClientPublicKey)
	if err != nil {
		return fmt.Errorf("error verifying client signature: %w", err)
	}
	if !ok {
		return errors.New("client's signature is invalid")
	}

	return nil
}

func instrument(name string, fn http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(fn, fmt.Sprintf("publicapi/%s", name))
}
