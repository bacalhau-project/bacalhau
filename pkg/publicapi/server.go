package publicapi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/requestornode"
	"github.com/filecoin-project/bacalhau/pkg/storage"
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

type submitRequest struct {
	Spec *executor.JobSpec `json:"spec"`
	Deal *executor.JobDeal `json:"deal"`
	// optional base64 encoded tar file that the api server will pin to ipfs for
	// you (the client), NOT part of the spec so we don't flood libp2p with
	// these files, max 10mb
	Context string `json:"context,omitempty"`
}

type submitResponse struct {
	Job *executor.Job `json:"job"`
}

func (apiServer *APIServer) submit(res http.ResponseWriter, req *http.Request) {
	var submitReq submitRequest
	if err := json.NewDecoder(req.Body).Decode(&submitReq); err != nil {
		log.Debug().Msgf("====> Decode submitReq error: %s", err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := job.VerifyJob(submitReq.Spec, submitReq.Deal); err != nil {
		log.Debug().Msgf("====> VerifyJob error: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if submitReq.Context != "" {
		// TODO: gc pinned contexts
		// TODO:
		//  * base64 decode submitReq.Context

		decoded, err := base64.StdEncoding.DecodeString(submitReq.Context)
		if err != nil {
			log.Debug().Msgf("====> DecodeContext error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//  * write decoded base64 to .tar file

		// create tmp dir
		tmpDir, err := ioutil.TempDir("", "bacalhau-pin-context-")
		if err != nil {
			log.Debug().Msgf("====> Create tmp dir error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// untar tmpDir/context.tar

		tarReader := bytes.NewReader(decoded)
		err = decompress(tarReader, filepath.Join(tmpDir, "context"))
		if err != nil {
			log.Debug().Msgf("====> Decompress error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//  * untar into directory
		//  * call apiServer.Node.PinContext with directory name
		cid, err := apiServer.Node.PinContext(filepath.Join(tmpDir, "context"))
		if err != nil {
			log.Debug().Msgf("====> PinContext error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		submitReq.Spec.Inputs = append(submitReq.Spec.Inputs, storage.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Cid:    cid,
			Path:   "/job",
		})
	}

	j, err := apiServer.Node.Transport.SubmitJob(req.Context(),
		submitReq.Spec, submitReq.Deal)
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

func instrument(name string, fn http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(fn, fmt.Sprintf("publicapi/%s", name))
}

// check for path traversal and correct forward slashes
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
