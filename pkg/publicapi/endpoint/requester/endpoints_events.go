package requester

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
)

// events godoc
//
//	@ID						pkg/requester/publicapi/events
//	@Summary				Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
//	@Description.markdown	endpoints_events
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					EventsRequest	body		apimodels.EventsRequest	true	"Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field."
//	@Success				200				{object}	apimodels.EventsResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/api/v1/requester/events [post]
//
//nolint:lll
//nolint:dupl
func (s *Endpoint) events(c echo.Context) error {
	var eventsReq apimodels.EventsRequest

	if err := c.Bind(&eventsReq); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	c.Response().Header().Set(apimodels.HTTPHeaderClientID, eventsReq.ClientID)
	c.Response().Header().Set(apimodels.HTTPHeaderJobID, eventsReq.JobID)

	ctx := c.Request().Context()
	events, err := s.jobStore.GetJobHistory(ctx, eventsReq.JobID, eventsReq.Options)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	legacyEvents := make([]model.JobHistory, len(events))
	for i := range events {
		legacyEvents[i] = *legacy.ToLegacyJobHistory(&events[i])
	}

	return c.JSON(http.StatusOK, apimodels.EventsResponse{
		Events: legacyEvents,
	})
}
