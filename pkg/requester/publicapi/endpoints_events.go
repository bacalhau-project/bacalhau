package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
)

type eventsRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	JobID    string `json:"job_id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
}

type eventsResponse struct {
	Events []model.JobHistory `json:"events"`
}

// events godoc
//
//	@ID						pkg/requester/publicapi/events
//	@Summary				Returns the events related to the job-id passed in the body payload. Useful for troubleshooting.
//	@Description.markdown	endpoints_events
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					eventsRequest	body		eventsRequest	true	"Request must specify a `client_id`. To retrieve your `client_id`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field."
//	@Success				200				{object}	eventsResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/requester/events [post]
//
//nolint:lll
//nolint:dupl
func (s *RequesterAPIServer) events(res http.ResponseWriter, req *http.Request) {
	var eventsReq eventsRequest
	if err := json.NewDecoder(req.Body).Decode(&eventsReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, eventsReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, eventsReq.JobID)

	ctx := req.Context()
	events, err := s.jobStore.GetJobHistory(ctx, eventsReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(eventsResponse{
		Events: events,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
