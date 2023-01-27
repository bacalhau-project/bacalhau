package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
)

type localEventsRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type localEventsResponse struct {
	LocalEvents []model.JobLocalEvent `json:"localEvents"`
}

// localEvents godoc
//
// @ID          pkg/requester/publicapi/localEvents
// @Summary     Returns the node's local events related to the job-id passed in the body payload. Useful for troubleshooting.
// @Description Local events (e.g. Selected, BidAccepted, Verified) are useful to track the progress of a job.
// @Tags        Job
// @Accept      json
// @Produce     json
// @Param       localEventsRequest body     localEventsRequest true " "
// @Success     200                {object} localEventsResponse
// @Failure     400                {object} string
// @Failure     500                {object} string
// @Router      /requester/local_events [post]
//
//nolint:dupl
func (s *RequesterAPIServer) localEvents(res http.ResponseWriter, req *http.Request) {
	var eventsReq localEventsRequest
	if err := json.NewDecoder(req.Body).Decode(&eventsReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, eventsReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, eventsReq.JobID)

	events, err := s.localDB.GetJobLocalEvents(req.Context(), eventsReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(localEventsResponse{
		LocalEvents: events,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
