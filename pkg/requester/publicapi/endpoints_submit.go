package publicapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/filecoin-project/bacalhau/pkg/system"
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

// Submit godoc
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
func (s *RequesterAPIServer) Submit(res http.ResponseWriter, req *http.Request) {
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

	j, err := s.requester.SubmitJob(
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
