package publicapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/targzip"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
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
		http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
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

		tmpDir, err := os.MkdirTemp("", "bacalhau-pin-context-")
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("====> Create tmp dir error: %s", err)
			errorResponse := bacerrors.ErrorToErrorResponse(err)
			http.Error(res, errorResponse, http.StatusInternalServerError)
			return
		}

		tarReader := bytes.NewReader(decoded)
		err = targzip.Decompress(tarReader, filepath.Join(tmpDir, "context"))
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
	span.SetAttributes(attribute.String(model.TracerAttributeNameJobID, j.ID))

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(submitResponse{
		Job: j,
	})
	if err != nil {
		http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusInternalServerError)
		return
	}
}
