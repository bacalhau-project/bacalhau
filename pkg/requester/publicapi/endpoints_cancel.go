package publicapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type cancelRequest struct {
	JobID    string `json:"id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	ClientID string `json:"ClientID,omitempty" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	Reason   string `json:"reason" example:"Canceled at user request"`
}

type CancelRequest = cancelRequest

type cancelResponse struct {
	State *model.JobState `json:"state"`
}

// cancel godoc
//
//	@ID						pkg/requester/publicapi/cancel
//	@Summary				Cancels the job with the job-id specified in the body payload.
//	@Description.markdown	endpoints_cancel
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					cancelRequest	body		cancelRequest	true	" "
//	@Success				200				{object}	cancelResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/requester/cancel [post]
func (s *RequesterAPIServer) cancel(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var cancelReq CancelRequest
	if err := json.NewDecoder(req.Body).Decode(&cancelReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, cancelReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, cancelReq.JobID)
	ctx = system.AddJobIDToBaggage(ctx, cancelReq.JobID)

	_, err := s.requester.CancelJob(ctx, requester.CancelJobRequest{
		JobID:         cancelReq.JobID,
		Reason:        cancelReq.Reason,
		UserTriggered: true,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Stuff

	jobState, err := getJobStateFromJobID(ctx, s, cancelReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(cancelResponse{
		State: &jobState,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getJobStateFromJobID(ctx context.Context, apiServer *RequesterAPIServer, jobID string) (model.JobState, error) {
	return apiServer.jobStore.GetJobState(ctx, jobID)
}
