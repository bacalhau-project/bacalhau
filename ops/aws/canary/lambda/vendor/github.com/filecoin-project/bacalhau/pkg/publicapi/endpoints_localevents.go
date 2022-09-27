package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type localEventsRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type localEventsResponse struct {
	LocalEvents []model.JobLocalEvent `json:"localEvents"`
}

//nolint:dupl
func (apiServer *APIServer) localEvents(res http.ResponseWriter, req *http.Request) {
	var eventsReq localEventsRequest
	if err := json.NewDecoder(req.Body).Decode(&eventsReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	events, err := apiServer.Controller.GetJobLocalEvents(req.Context(), eventsReq.JobID)
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
