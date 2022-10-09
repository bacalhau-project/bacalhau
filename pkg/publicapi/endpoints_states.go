package publicapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
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
	ctx, span := system.GetSpanFromRequest(req, "pkg/publicapi/states")
	defer span.End()

	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, stateReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, stateReq.JobID)
	ctx = system.AddJobIDToBaggage(ctx, stateReq.JobID)

	js, err := getJobStateFromRequest(ctx, apiServer, stateReq)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(stateResponse{
		State: js,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getJobStateFromRequest(ctx context.Context, apiServer *APIServer, stateReq stateRequest) (model.JobState, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publicapi/getJobStateFromRequest")
	defer span.End()

	return apiServer.localdb.GetJobState(ctx, stateReq.JobID)
}
