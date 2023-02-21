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
	oteltrace "go.opentelemetry.io/otel/trace"
)

type submitRequest struct {
	// The data needed to submit and run a job on the network:
	JobCreatePayload *json.RawMessage `json:"job_create_payload" validate:"required"`

	// A base64-encoded signature of the data, signed by the client:
	ClientSignature string `json:"signature" validate:"required"`

	// The base64-encoded public key of the client:
	ClientPublicKey string `json:"client_public_key" validate:"required"`
}

type submitResponse struct {
	Job *model.Job `json:"job"`
}

// submit godoc
//
//	@ID						pkg/requester/publicapi/submit
//	@Summary				Submits a new job to the network.
//	@Description.markdown	endpoints_submit
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					submitRequest	body		submitRequest	true	" "
//	@Success				200				{object}	submitResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/requester/submit [post]
func (s *RequesterAPIServer) submit(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
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

	// first verify the signature on the raw bytes
	if err := verifyRequestSignature(*submitReq.JobCreatePayload, submitReq.ClientSignature, submitReq.ClientPublicKey); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> VerifyRequestSignature error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	// then decode the job create payload
	var jobCreatePayload model.JobCreatePayload
	if err := json.Unmarshal(*submitReq.JobCreatePayload, &jobCreatePayload); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> Decode JobCreatePayload error: %s", err)
		http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, jobCreatePayload.ClientID)

	if err := verifySignedJobRequest(jobCreatePayload.ClientID, submitReq.ClientSignature, submitReq.ClientPublicKey); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> verifySignedJobRequest error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	if err := job.VerifyJobCreatePayload(ctx, &jobCreatePayload); err != nil {
		log.Ctx(ctx).Debug().Msgf("====> VerifyJobCreate error: %s", err)
		errorResponse := bacerrors.ErrorToErrorResponse(err)
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	j, err := s.requester.SubmitJob(
		ctx,
		jobCreatePayload,
	)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, j.Metadata.ID)
	ctx = system.AddJobIDToBaggage(ctx, j.Metadata.ID)
	system.AddJobIDFromBaggageToSpan(ctx, oteltrace.SpanFromContext(ctx))

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
