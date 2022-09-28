package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type eventsRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type eventsResponse struct {
	Events []model.JobEvent `json:"events"`
}

//nolint:dupl
func (apiServer *APIServer) events(res http.ResponseWriter, req *http.Request) {
	var eventsReq eventsRequest
	if err := json.NewDecoder(req.Body).Decode(&eventsReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	events, err := apiServer.localdb.GetJobEvents(req.Context(), eventsReq.JobID)
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
