package publicapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
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
func (apiServer *APIServer) ListenAndServe(ctx context.Context) error {
	hostID, err := apiServer.Node.Transport.HostID(ctx)
	if err != nil {
		log.Error().Msgf("Error fetching node's host ID: %s", err)
		return err
	}

	sm := http.NewServeMux()
	sm.Handle("/list", http.HandlerFunc(apiServer.list))
	sm.Handle("/submit", http.HandlerFunc(apiServer.submit))
	sm.Handle("/healthz", http.HandlerFunc(apiServer.healthz))
	sm.Handle("/logz", http.HandlerFunc(apiServer.logz))
	sm.Handle("/varz", http.HandlerFunc(apiServer.varz))
	sm.Handle("/livez", http.HandlerFunc(apiServer.livez))
	sm.Handle("/readyz", http.HandlerFunc(apiServer.readyz))

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
		Handler: sm,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx // TODO: handle trace ID stuff here
		},
	}

	log.Debug().Msgf(
		"API server listening for host %s on %s...", hostID, srv.Addr)
	return srv.ListenAndServe()
}

func (apiServer *APIServer) livez(res http.ResponseWriter, req *http.Request) {
	// Extremely simple liveness check (should be fine to be public / no-auth)
	log.Debug().Msg("Received OK request")
	res.Header().Add("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	_, err := res.Write([]byte("OK"))
	if err != nil {
		log.Warn().Msg("Error writing body for OK request.")
	}
}

func (apiServer *APIServer) logz(res http.ResponseWriter, req *http.Request) {
	log.Debug().Msg("Received logz request")
	res.Header().Add("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	fileOutput, err := TailFile(100, "/tmp/ipfs.log")
	if err != nil {
		missingLogFileMsg := "File not found at /tmp/ipfs.log"
		log.Warn().Msgf(missingLogFileMsg)
		_, err = res.Write([]byte("File not found at /tmp/ipfs.log"))
		if err != nil {
			log.Warn().Msgf("Could not write response body for missing log file to response.")
		}
		return
	}
	_, err = res.Write(fileOutput)
	if err != nil {
		log.Warn().Msg("Error writing body for logz request.")
	}

}

func (apiServer *APIServer) readyz(res http.ResponseWriter, req *http.Request) {
	log.Debug().Msg("Received readyz request.")
	// TODO: Add checker for queue that this node can accept submissions
	isReady := true

	res.Header().Add("Content-Type", "text/plain")
	if !isReady {
		res.WriteHeader(http.StatusServiceUnavailable)
	}
	res.WriteHeader(http.StatusOK)
	_, err := res.Write([]byte("READY"))
	if err != nil {
		log.Warn().Msg("Error writing body for readyz request.")
	}

}


func GenerateHealthData() types.HealthInfo {

	var healthInfo types.HealthInfo

	// Generating all, free, used amounts for each - in case these are different mounts, they'll have different
	// All and Free values, if they're all on the same machine, then those values should be the same
	// If "All" is 0, that means the directory does not exist
	healthInfo.DiskFreeSpace.IPFSMount = MountUsage("/data/ipfs")
	healthInfo.DiskFreeSpace.ROOT = MountUsage("/")
	healthInfo.DiskFreeSpace.TMP = MountUsage("/tmp")

	return healthInfo
}

func (apiServer *APIServer) healthz(res http.ResponseWriter, req *http.Request) {
	// TODO: A list of health information. Should require authing (of some kind)
	log.Debug().Msg("Received healthz request.")
	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	// Ideas:
	// CPU usage

	healthInfo := GenerateHealthData()
	healthJsonBlob, _ := json.Marshal(healthInfo)

	_, err := res.Write([]byte(healthJsonBlob))
	if err != nil {
		log.Warn().Msg("Error writing body for healthz request.")
	}
}

func (apiServer *APIServer) varz(res http.ResponseWriter, req *http.Request) {
	// TODO: Fill in with the configuration settings for this node
	res.WriteHeader(http.StatusOK)
	res.Header().Add("Content-Type", "application/json")

	_, err := res.Write([]byte("{}"))
	if err != nil {
		log.Warn().Msg("Error writing body for varz request.")
	}
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
