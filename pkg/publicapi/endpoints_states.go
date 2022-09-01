package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type stateRequest struct {
	ClientID string `json:"client_id"`
	JobID    string `json:"job_id"`
}

type stateResponse struct {
	State model.JobState `json:"state"`
}

func (apiServer *APIServer) states(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/states")
	defer span.End()
	t := system.GetTracer()

	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	_, getJobSpan := t.Start(ctx, "gettingjobstate")
	jobState, err := apiServer.Controller.GetJobState(ctx, stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	getJobSpan.End()

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(stateResponse{
		State: jobState,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
