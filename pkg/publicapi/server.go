package publicapi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	sync "github.com/RobinUS2/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func init() {
	sync.SetGlobalOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Enabled:   true,
		Id:        "<UNKNOWN>",
	})
}

const ServerReadHeaderTimeout = 10 * time.Second

// APIServer configures a node's public REST API.
type APIServer struct {
	Controller  *controller.Controller
	Verifiers   map[verifier.VerifierType]verifier.Verifier
	Host        string
	Port        int
	componentMu sync.Mutex
}

// NewServer returns a new API server for a requester node.
func NewServer(
	host string,
	port int,
	c *controller.Controller,
	verifiers map[verifier.VerifierType]verifier.Verifier,
) *APIServer {
	a := &APIServer{
		Controller: c,
		Verifiers:  verifiers,
		Host:       host,
		Port:       port,
	}
	a.componentMu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "APIServer.componentMu",
	})
	return a
	a.componentMu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "APIServer.componentMu",
	})
	return a
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *APIServer) GetURI() string {
	return fmt.Sprintf("http://%s:%d", apiServer.Host, apiServer.Port)
}

// ListenAndServe listens for and serves HTTP requests against the API server.
func (apiServer *APIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	hostID, err := apiServer.Controller.HostID(ctx)
	if err != nil {
		return err
	}
	sm := http.NewServeMux()
	sm.Handle("/list", instrument("list", apiServer.list))
	sm.Handle("/states", instrument("states", apiServer.states))
	sm.Handle("/results", instrument("results", apiServer.results))
	sm.Handle("/events", instrument("events", apiServer.events))
	sm.Handle("/local_events", instrument("local_events", apiServer.localEvents))
	sm.Handle("/id", instrument("id", apiServer.id))
	sm.Handle("/peers", instrument("peers", apiServer.peers))
	sm.Handle("/submit", instrument("submit", apiServer.submit))
	sm.Handle("/version", instrument("version", apiServer.version))
	sm.Handle("/healthz", instrument("healthz", apiServer.healthz))
	sm.Handle("/logz", instrument("logz", apiServer.logz))
	sm.Handle("/varz", instrument("varz", apiServer.varz))
	sm.Handle("/livez", instrument("livez", apiServer.livez))
	sm.Handle("/readyz", instrument("readyz", apiServer.readyz))
	sm.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Handler:           sm,
		Addr:              fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
		ReadHeaderTimeout: ServerReadHeaderTimeout,
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
	Jobs map[string]executor.Job `json:"jobs"`
}

type stateRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type stateResponse struct {
	State executor.JobState `json:"state"`
}

type resultsRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type resultsResponse struct {
	Results []storage.StorageSpec `json:"results"`
}

type eventsRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type eventsResponse struct {
	Events []executor.JobEvent `json:"events"`
}

type localEventsRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type localEventsResponse struct {
	LocalEvents []executor.JobLocalEvent `json:"localEvents"`
}

type versionRequest struct {
	ClientID string `json:"client_id"`
}

type versionResponse struct {
	VersionInfo *executor.VersionInfo `json:"version_info"`
}

func (apiServer *APIServer) id(res http.ResponseWriter, req *http.Request) {
	switch apiTransport := apiServer.Controller.GetTransport().(type) { //nolint:gocritic
	case *libp2p.LibP2PTransport:
		id, err := apiTransport.HostID(context.Background())
		if err != nil {
			http.Error(res, fmt.Sprintf("Error getting id: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)
		err = json.NewEncoder(res).Encode(id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

func (apiServer *APIServer) peers(res http.ResponseWriter, req *http.Request) {
	// switch on apiTransport type to get the right method
	// we need to use a switch here because we want to look at .(type)
	// ^ that is a note for you gocritic
	switch apiTransport := apiServer.Controller.GetTransport().(type) { //nolint:gocritic
	case *libp2p.LibP2PTransport:
		peers, err := apiTransport.GetPeers(context.Background())
		if err != nil {
			http.Error(res, fmt.Sprintf("Error getting peers: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		// write response to res
		res.WriteHeader(http.StatusOK)
		err = json.NewEncoder(res).Encode(peers)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	http.Error(res, "Not a libp2p transport", http.StatusInternalServerError)
}

func (apiServer *APIServer) list(res http.ResponseWriter, req *http.Request) {
	var listReq listRequest
	if err := json.NewDecoder(req.Body).Decode(&listReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	list, err := apiServer.Controller.GetJobs(req.Context(), localdb.JobQuery{})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	rawJobs := map[string]executor.Job{}

	for _, listJob := range list { //nolint:gocritic
		rawJobs[listJob.ID] = listJob
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(listResponse{
		Jobs: rawJobs,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *APIServer) states(res http.ResponseWriter, req *http.Request) {
	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	jobState, err := apiServer.Controller.GetJobState(req.Context(), stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(stateResponse{
		State: jobState,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *APIServer) results(res http.ResponseWriter, req *http.Request) {
	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	job, err := apiServer.Controller.GetJob(req.Context(), stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	verifier, err := apiServer.getVerifier(req.Context(), job.Spec.Verifier)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	results, err := verifier.GetJobResultSet(req.Context(), stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(resultsResponse{
		Results: results,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

//nolint:dupl
func (apiServer *APIServer) events(res http.ResponseWriter, req *http.Request) {
	var eventsReq eventsRequest
	if err := json.NewDecoder(req.Body).Decode(&eventsReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	events, err := apiServer.Controller.GetJobEvents(req.Context(), eventsReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(eventsResponse{
		Events: events,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

//nolint:dupl
func (apiServer *APIServer) localEvents(res http.ResponseWriter, req *http.Request) {
	var eventsReq localEventsRequest
	if err := json.NewDecoder(req.Body).Decode(&eventsReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	events, err := apiServer.Controller.GetJobLocalEvents(req.Context(), eventsReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(localEventsResponse{
		LocalEvents: events,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (apiServer *APIServer) version(res http.ResponseWriter, req *http.Request) {
	var versionReq versionRequest
	err := json.NewDecoder(req.Body).Decode(&versionReq)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(versionResponse{
		VersionInfo: version.Get(),
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

type submitRequest struct {
	// The data needed to submit and run a job on the network:
	Data executor.JobCreatePayload `json:"data"`

	// A base64-encoded signature of the data, signed by the client:
	ClientSignature string `json:"signature"`

	// The base64-encoded public key of the client:
	ClientPublicKey string `json:"client_public_key"`
}

type submitResponse struct {
	Job executor.Job `json:"job"`
}

func (apiServer *APIServer) submit(res http.ResponseWriter, req *http.Request) {
	var submitReq submitRequest
	if err := json.NewDecoder(req.Body).Decode(&submitReq); err != nil {
		log.Debug().Msgf("====> Decode submitReq error: %s", err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := verifySubmitRequest(&submitReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := job.VerifyJob(submitReq.Data.Spec, submitReq.Data.Deal); err != nil {
		log.Debug().Msgf("====> VerifyJob error: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// If we have a build context, pin it to IPFS and mount it in the job:
	if submitReq.Data.Context != "" {
		// TODO: gc pinned contexts
		decoded, err := base64.StdEncoding.DecodeString(submitReq.Data.Context)
		if err != nil {
			log.Debug().Msgf("====> DecodeContext error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpDir, err := ioutil.TempDir("", "bacalhau-pin-context-")
		if err != nil {
			log.Debug().Msgf("====> Create tmp dir error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		tarReader := bytes.NewReader(decoded)
		err = decompress(tarReader, filepath.Join(tmpDir, "context"))
		if err != nil {
			log.Debug().Msgf("====> Decompress error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		cid, err := apiServer.Controller.PinContext(req.Context(), filepath.Join(tmpDir, "context"))
		if err != nil {
			log.Debug().Msgf("====> PinContext error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// NOTE(luke): we could do some kind of storage multiaddr here, e.g.:
		//               --cid ipfs:abc --cid filecoin:efg
		submitReq.Data.Spec.Contexts = append(submitReq.Data.Spec.Contexts, storage.StorageSpec{
			Engine: storage.StorageSourceIPFS,
			Cid:    cid,
			Path:   "/job",
		})
	}

	j, err := apiServer.Controller.SubmitJob(
		req.Context(),
		submitReq.Data,
	)

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

func (apiServer *APIServer) getVerifier(ctx context.Context, typ verifier.VerifierType) (verifier.Verifier, error) {
	apiServer.componentMu.Lock()
	defer apiServer.componentMu.Unlock()

	if _, ok := apiServer.Verifiers[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", typ.String())
	}

	v := apiServer.Verifiers[typ]
	installed, err := v.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	return v, nil
}

func verifySubmitRequest(req *submitRequest) error {
	if req.Data.ClientID == "" {
		return errors.New("job deal must contain a client ID")
	}
	if req.ClientSignature == "" {
		return errors.New("client's signature is required")
	}
	if req.ClientPublicKey == "" {
		return errors.New("client's public key is required")
	}

	// Check that the client's public key matches the client ID:
	ok, err := system.PublicKeyMatchesID(req.ClientPublicKey, req.Data.ClientID)
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

	err = system.Verify(jsonData, req.ClientSignature, req.ClientPublicKey)
	if err != nil {
		return fmt.Errorf("client's signature is invalid: %w", err)
	}

	return nil
}

func instrument(name string, fn http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(fn, fmt.Sprintf("publicapi/%s", name))
}

// check for path traversal and correct forward slashes
//
//nolint:deadcode,unused
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

// Sanitize archive file pathing from "G305: Zip Slip vulnerability"
func SanitizeArchivePath(d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}

//nolint:unused,deadcode
func decompress(src io.Reader, dst string) error {
	// ungzip
	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	// untar
	tr := tar.NewReader(zr)

	// uncompress each element
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		target := header.Name

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", target)
		}

		// add dst + re-format slashes according to system
		target, err = SanitizeArchivePath(dst, header.Name)
		if err != nil {
			return err
		}
		// if no join is needed, replace with ToSlash:
		// target = filepath.ToSlash(header.Name)

		// check the type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it (with 0755 permission)
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil { //nolint:gomnd
					return err
				}
			}
		// if it's a file create it (with same permission)
		case tar.TypeReg:
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents (max 10MB per file!)
			// TODO: error if files are too big, rather than silently truncating them :-O
			if _, err := io.CopyN(fileToWrite, tr, 10*1024*1024); err != nil { //nolint:gomnd
				log.Debug().Msgf("CopyN err is %s", err)
				// io.EOF is expected
				if err != io.EOF {
					return err
				}
			}
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			fileToWrite.Close()
		}
	}

	//
	return nil
}
