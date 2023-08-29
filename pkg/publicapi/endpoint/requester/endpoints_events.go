package requester

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/go-chi/render"
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
func (s *Endpoint) events(res http.ResponseWriter, req *http.Request) {
	var eventsReq apimodels.EventsRequest
	if err := render.DecodeJSON(req.Body, &eventsReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(apimodels.HTTPHeaderClientID, eventsReq.ClientID)
	res.Header().Set(apimodels.HTTPHeaderJobID, eventsReq.JobID)

	ctx := req.Context()
	events, err := s.jobStore.GetJobHistory(ctx, eventsReq.JobID, eventsReq.Options)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	legacyEvents := make([]model.JobHistory, len(events))
	for i := range events {
		legacyEvents[i] = *legacy.ToLegacyJobHistory(&events[i])
	}

	response := apimodels.EventsResponse{
		Events: legacyEvents,
	}
	render.JSON(res, req, response)
}
