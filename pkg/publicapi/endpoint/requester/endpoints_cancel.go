package requester

import (
	"fmt"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/signatures"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

// @ID				pkg/requester/publicapi/cancel
// @Summary		Cancels the job with the job-id specified in the body payload.
// @Description	Cancels a job specified by `id` as long as that job belongs to `client_id`.
// @Description	Returns the current jobstate after the cancel request has been processed.
// @Tags			Job
// @Accept			json
// @Produce		json
// @Param			CancelRequest	body		legacymodels.CancelRequest	true	" "
// @Success		200				{object}	legacymodels.CancelResponse
// @Failure		400				{object}	string
// @Failure		401				{object}	string
// @Failure		403				{object}	string
// @Failure		500				{object}	string
// @Router			/api/v1/requester/cancel [post]
func (s *Endpoint) cancel(c echo.Context) error {
	ctx := c.Request().Context()
	jobCancelPayload, err := signatures.UnmarshalSigned[model.JobCancelPayload](ctx, c.Request().Body)
	if err != nil {
		publicapi.HTTPError(c, err, http.StatusBadRequest)
		return nil
	}

	c.Response().Header().Set(apimodels.HTTPHeaderClientID, jobCancelPayload.ClientID)
	ctx = system.AddJobIDToBaggage(ctx, jobCancelPayload.ClientID)

	// Get the job, check it exists and check it belongs to the same client
	job, err := s.jobStore.GetJob(ctx, jobCancelPayload.JobID)
	if err != nil {
		publicapi.HTTPError(c, errors.Wrap(err, "missing job"), http.StatusNotFound)
		return nil
	}

	// We can compare the payload's client ID against the existing job's metadata
	// as we have confirmed the public key that the request was signed with matches
	// the client ID the request claims.
	if job.Namespace != jobCancelPayload.ClientID {
		err = fmt.Errorf("mismatched ClientIDs for cancel, existing job: %s and cancel request: %s",
			job.Namespace, jobCancelPayload.ClientID)
		publicapi.HTTPError(c, err, http.StatusUnauthorized)
		return nil
	}

	_, err = s.requester.CancelJob(ctx, requester.CancelJobRequest{
		JobID:         jobCancelPayload.JobID,
		Reason:        jobCancelPayload.Reason,
		UserTriggered: true,
	})
	if err != nil {
		publicapi.HTTPError(c, err, http.StatusInternalServerError)
		return nil
	}

	jobState, err := legacy.GetJobState(ctx, s.jobStore, jobCancelPayload.JobID)
	if err != nil {
		publicapi.HTTPError(c, err, http.StatusInternalServerError)
		return nil
	}

	c.Response().Header().Set(apimodels.HTTPHeaderClientID, jobCancelPayload.ClientID)
	c.Response().Header().Set(apimodels.HTTPHeaderJobID, jobCancelPayload.JobID)
	return c.JSON(http.StatusOK, legacymodels.CancelResponse{
		State: &jobState,
	})
}
