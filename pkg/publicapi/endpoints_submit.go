package publicapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/targzip"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

type submitRequest struct {
	// The data needed to submit and run a job on the network:
	JobCreatePayload model.JobCreatePayload `json:"job_create_payload" validate:"required"`

	// A base64-encoded signature of the data, signed by the client:
	ClientSignature string `json:"signature" validate:"required"`

	// The base64-encoded public key of the client:
	ClientPublicKey string `json:"client_public_key" validate:"required"`
}

type submitResponse struct {
	Job *model.Job `json:"job"`
}

// submit godoc
// @ID                   pkg/apiServer.submit
// @Summary              Submits a new job to the network.
// @Description.markdown endpoints_submit
// @Tags                 Job
// @Accept               json
// @Produce              json
// @Param                submitRequest body     submitRequest true " "
// @Success              200           {object} submitResponse
// @Failure              400           {object} string
// @Failure              500           {object} string
// @Router               /submit [post]
func (apiServer *APIServer) submit(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "pkg/apiServer.submit")
	defer span.End()

	if otherJobID := req.Header.Get("X-Bacalhau-Job-ID"); otherJobID != "" {
		err := fmt.Errorf("rejecting job because HTTP header X-Bacalhau-Job-ID was set")
		log.Ctx(ctx).Info().Str("X-Bacalhau-Job-ID", otherJobID).Err(err).Send()
		http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
		return
	}

	var submitReq submitRequest
	if err := json.NewDecoder(req.Body).Decode(&submitReq); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> Decode submitReq error: %s", err)
		http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, submitReq.JobCreatePayload.ClientID)

	if err := verifySubmitRequest(&submitReq); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> VerifySubmitRequest error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	if err := job.VerifyJobCreatePayload(ctx, &submitReq.JobCreatePayload); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> VerifyJobCreate error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	// If we have a build context, pin it to IPFS and mount it in the job:
	if submitReq.JobCreatePayload.Context != "" {
		spec, err := apiServer.saveInlineTarball(ctx, submitReq.JobCreatePayload.Context)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error saving build context")
			http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusInternalServerError)
			return
		}
		submitReq.JobCreatePayload.Spec.Contexts = append(
			submitReq.JobCreatePayload.Spec.Contexts,
			spec,
		)
	}

	j, err := apiServer.Requester.SubmitJob(
		ctx,
		submitReq.JobCreatePayload,
	)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, j.Metadata.ID)
	span.SetAttributes(attribute.String(model.TracerAttributeNameJobID, j.Metadata.ID))

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

func (apiServer *APIServer) saveInlineTarball(ctx context.Context, base64tar string) (model.StorageSpec, error) {
	// TODO: gc pinned contexts
	decoded, err := base64.StdEncoding.DecodeString(base64tar)
	if err != nil {
		return model.StorageSpec{}, errors.Wrap(err, "error base64 decoding context")
	}

	tmpDir, err := os.MkdirTemp("", "bacalhau-pin-context-")
	if err != nil {
		return model.StorageSpec{}, errors.Wrap(err, "error creating temp dir")
	}

	tarReader := bytes.NewReader(decoded)
	err = targzip.Decompress(tarReader, filepath.Join(tmpDir, "context"))
	if err != nil {
		return model.StorageSpec{}, errors.Wrap(err, "error decompressing context")
	}

	// write the "context" for a job to storage
	// this is used to upload code files
	// we presently just fix on ipfs to do this
	ipfsStorage, err := apiServer.StorageProviders.GetStorage(ctx, model.StorageSourceIPFS)
	if err != nil {
		return model.StorageSpec{}, errors.Wrap(err, "error getting storage provider")
	}

	result, err := ipfsStorage.Upload(ctx, filepath.Join(tmpDir, "context"))
	if err != nil {
		return model.StorageSpec{}, errors.Wrap(err, "error uploading context to IPFS")
	}

	// NOTE(luke): we could do some kind of storage multiaddr here, e.g.:
	//               --cid ipfs:abc --cid filecoin:efg
	return model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           result.CID,
		Path:          "/job",
	}, nil
}
