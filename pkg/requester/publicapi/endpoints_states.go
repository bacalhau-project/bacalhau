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
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	JobID    string `json:"job_id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
}

type stateResponse struct {
	State model.JobState `json:"state"`
}

// states godoc
//
//	@ID						pkg/requester/publicapi/states
//	@Summary				Returns the state of the job-id specified in the body payload.
//	@Description.markdown	endpoints_states
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					stateRequest	body		stateRequest	true	" "
//	@Success				200				{object}	stateResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/requester/states [post]
func (s *RequesterAPIServer) states(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, stateReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, stateReq.JobID)
	ctx = system.AddJobIDToBaggage(ctx, stateReq.JobID)

	js, err := getJobStateFromRequest(ctx, s, stateReq)
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

func getJobStateFromRequest(ctx context.Context, apiServer *RequesterAPIServer, stateReq stateRequest) (model.JobState, error) {
	return apiServer.jobStore.GetJobState(ctx, stateReq.JobID)
}
