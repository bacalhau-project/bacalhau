package publicapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type submitRequest = publicapi.SignedRequest[v1beta2.JobCreatePayload] //nolint:unused // Swagger wants this

type submitResponse struct {
	Job *v1beta2.Job `json:"job"`
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
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	jobCreatePayload, err := publicapi.UnmarshalSigned[v1beta2.JobCreatePayload](ctx, req.Body)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	payload := model.ConvertV1beta2JobCreatePayload(jobCreatePayload)

	if err := job.VerifyJobCreatePayload(ctx, &payload); err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	j, err := s.requester.SubmitJob(ctx, payload)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, j.Metadata.ID)
	ctx = system.AddJobIDToBaggage(ctx, j.Metadata.ID)
	system.AddJobIDFromBaggageToSpan(ctx, oteltrace.SpanFromContext(ctx))

	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusInternalServerError)
		return
	}

	betaJob := model.ConvertJobToV1beta2(*j)

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(submitResponse{Job: &betaJob})
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusInternalServerError)
		return
	}
}
