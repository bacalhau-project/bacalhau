package requester

import (
	"encoding/json"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// states godoc
//
//	@ID						pkg/requester/publicapi/states
//	@Summary				Returns the state of the job-id specified in the body payload.
//	@Description.markdown	endpoints_states
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					StateRequest	body		apimodels.StateRequest	true	" "
//	@Success				200				{object}	apimodels.StateResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/api/v1/requester/states [post]
func (s *Endpoint) states(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var stateReq apimodels.StateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(apimodels.HTTPHeaderClientID, stateReq.ClientID)
	res.Header().Set(apimodels.HTTPHeaderJobID, stateReq.JobID)
	ctx = system.AddJobIDToBaggage(ctx, stateReq.JobID)

	js, err := legacy.GetJobState(ctx, s.jobStore, stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(apimodels.StateResponse{
		State: js,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
