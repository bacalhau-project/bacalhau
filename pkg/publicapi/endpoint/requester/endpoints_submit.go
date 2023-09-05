package requester

import (
	"fmt"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/signatures"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/labstack/echo/v4"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// submit godoc
//
//	@ID						pkg/requester/publicapi/submit
//	@Summary				Submits a new job to the network.
//	@Description.markdown	endpoints_submit
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					SubmitRequest	body		legacy.SubmitRequest	true	" "
//	@Success				200				{object}	legacy.SubmitResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/api/v1/requester/submit [post]
func (s *Endpoint) submit(c echo.Context) error {
	ctx := c.Request().Context()

	if otherJobID := c.Request().Header.Get("X-Bacalhau-Job-ID"); otherJobID != "" {
		err := fmt.Errorf("rejecting job because HTTP header X-Bacalhau-Job-ID was set")
		publicapi.HTTPError(c, err, http.StatusBadRequest)
		return nil
	}

	jobCreatePayload, err := signatures.UnmarshalSigned[model.JobCreatePayload](ctx, c.Request().Body)
	if err != nil {
		publicapi.HTTPError(c, err, http.StatusBadRequest)
		return nil
	}

	if err := job.VerifyJobCreatePayload(ctx, &jobCreatePayload); err != nil {
		publicapi.HTTPError(c, err, http.StatusBadRequest)
		return nil
	}

	j, err := s.requester.SubmitJob(ctx, jobCreatePayload)
	if err != nil {
		publicapi.HTTPError(c, err, http.StatusInternalServerError)
		return nil
	}

	c.Response().Header().Set(apimodels.HTTPHeaderJobID, j.Metadata.ID)
	ctx = system.AddJobIDToBaggage(ctx, j.Metadata.ID)
	system.AddJobIDFromBaggageToSpan(ctx, oteltrace.SpanFromContext(ctx))

	return c.JSON(http.StatusOK, legacymodels.SubmitResponse{Job: j})
}
