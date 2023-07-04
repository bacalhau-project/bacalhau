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
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type submitRequest = publicapi.SignedRequest[model.JobCreatePayload] //nolint:unused // Swagger wants this

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
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	jobCreatePayload, err := publicapi.UnmarshalSigned[model.JobCreatePayload](ctx, req.Body)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	if jobCreatePayload.ClientID == "" {
		err := fmt.Errorf("ClientID is empty")
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	if jobCreatePayload.APIVersion == "" {
		err := fmt.Errorf("APIVersion is empty")
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	apiVersion, err := model.ParseAPIVersion(jobCreatePayload.APIVersion)
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	var spec model.Spec
	switch apiVersion {
	case model.V1beta2:
		var oldSpec v1beta2.Spec
		if err := json.Unmarshal(jobCreatePayload.Spec, &oldSpec); err != nil {
			publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
			return
		}
		spec = model.ConvertV1beta2Spec(oldSpec)
	case model.V1beta3:
		if err := json.Unmarshal(jobCreatePayload.Spec, &spec); err != nil {
			publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
			return
		}
	default:
		publicapi.HTTPError(ctx, res, fmt.Errorf("unhandled api version: %s", apiVersion), http.StatusInternalServerError)
		return
	}

	// the job has been migrated to the latest version
	submittedJob := model.Job{
		APIVersion: model.V1beta3.String(),
		Spec:       spec,
	}

	if err := job.VerifyJob(ctx, &submittedJob); err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusBadRequest)
		return
	}

	j, err := s.requester.SubmitJob(ctx, requester.SubmitJobRequest{
		ClientID: jobCreatePayload.ClientID,
		Job:      submittedJob,
	})
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, j.Metadata.ID)
	ctx = system.AddJobIDToBaggage(ctx, j.Metadata.ID)
	system.AddJobIDFromBaggageToSpan(ctx, oteltrace.SpanFromContext(ctx))

	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(submitResponse{Job: j})
	if err != nil {
		publicapi.HTTPError(ctx, res, err, http.StatusInternalServerError)
		return
	}
}
