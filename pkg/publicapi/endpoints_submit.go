package publicapi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type submitRequest struct {
	// The data needed to submit and run a job on the network:
	Data model.JobCreatePayload `json:"data"`

	// A base64-encoded signature of the data, signed by the client:
	ClientSignature string `json:"signature"`

	// The base64-encoded public key of the client:
	ClientPublicKey string `json:"client_public_key"`
}

type submitResponse struct {
	Job *model.Job `json:"job"`
}

func (apiServer *APIServer) submit(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "pkg/apiServer.submit")
	defer span.End()

	var submitReq submitRequest
	if err := json.NewDecoder(req.Body).Decode(&submitReq); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> Decode submitReq error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, submitReq.Data.ClientID)

	if err := verifySubmitRequest(&submitReq); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> VerifySubmitRequest error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	if err := job.VerifyJob(ctx, submitReq.Data.Job); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> VerifyJob error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	// If we have a build context, pin it to IPFS and mount it in the job:
	if submitReq.Data.Context != "" {
		// TODO: gc pinned contexts
		decoded, err := base64.StdEncoding.DecodeString(submitReq.Data.Context)
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("====> DecodeContext error: %s", err)
			errorResponse := bacerrors.ErrorToErrorResponse(err)
			http.Error(res, errorResponse, http.StatusInternalServerError)
			return
		}

		tmpDir, err := ioutil.TempDir("", "bacalhau-pin-context-")
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("====> Create tmp dir error: %s", err)
			errorResponse := bacerrors.ErrorToErrorResponse(err)
			http.Error(res, errorResponse, http.StatusInternalServerError)
			return
		}

		tarReader := bytes.NewReader(decoded)
		err = decompress(tarReader, filepath.Join(tmpDir, "context"))
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("====> Decompress error: %s", err)
			errorResponse := bacerrors.ErrorToErrorResponse(err)
			http.Error(res, errorResponse, http.StatusInternalServerError)
			return
		}

		// write the "context" for a job to storage
		// this is used to upload code files
		// we presently just fix on ipfs to do this
		ipfsStorage, err := apiServer.StorageProviders.GetStorage(ctx, model.StorageSourceIPFS)
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("====> GetStorage error: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		result, err := ipfsStorage.Upload(ctx, filepath.Join(tmpDir, "context"))
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("====> PinContext error: %s", err)
			errorResponse := bacerrors.ErrorToErrorResponse(err)
			http.Error(res, errorResponse, http.StatusInternalServerError)
			return
		}

		// NOTE(luke): we could do some kind of storage multiaddr here, e.g.:
		//               --cid ipfs:abc --cid filecoin:efg
		submitReq.Data.Job.Spec.Contexts = append(submitReq.Data.Job.Spec.Contexts, model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			CID:           result.CID,
			Path:          "/job",
		})
	}

	j, err := apiServer.Requester.SubmitJob(
		ctx,
		submitReq.Data,
	)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, j.ID)

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(submitResponse{
		Job: j,
	})
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusInternalServerError)
		return
	}
}

const worldReadOwnerWritePermission fs.FileMode = 0755

func decompress(src io.Reader, dst string) error {
	// ensure destination directory exists
	err := os.Mkdir(dst, worldReadOwnerWritePermission)
	if err != nil {
		return err
	}

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
				if err := os.MkdirAll(target, worldReadOwnerWritePermission); err != nil {
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

// check for path traversal and correct forward slashes
//
//nolint:unused
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
