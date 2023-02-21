package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/jobstore"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/filecoin-project/bacalhau/pkg/system"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type resultsRequest struct {
	ClientID string `json:"client_id" example:"ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51"`
	JobID    string `json:"job_id" example:"9304c616-291f-41ad-b862-54e133c0149e"`
}

type resultsResponse struct {
	Results []model.PublishedResult `json:"results"`
}

// results godoc
//
//	@ID						pkg/requester/publicapi/results
//	@Summary				Returns the results of the job-id specified in the body payload.
//	@Description.markdown	endpoints_results
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					stateRequest	body		stateRequest	true	" "
//	@Success				200				{object}	resultsResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/requester/results [post]
func (s *RequesterAPIServer) results(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var stateReq stateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(handlerwrapper.HTTPHeaderClientID, stateReq.ClientID)
	res.Header().Set(handlerwrapper.HTTPHeaderJobID, stateReq.JobID)

	ctx = system.AddJobIDToBaggage(ctx, stateReq.JobID)
	system.AddJobIDFromBaggageToSpan(ctx, oteltrace.SpanFromContext(ctx))

	stateResolver := jobstore.GetStateResolver(s.jobStore)
	results, err := stateResolver.GetResults(ctx, stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(resultsResponse{
		Results: results,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
