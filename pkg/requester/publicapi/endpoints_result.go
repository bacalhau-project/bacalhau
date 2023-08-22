package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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

	executions, err := s.jobStore.GetExecutions(ctx, stateReq.JobID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	results := make([]model.PublishedResult, 0)
	for _, execution := range executions {
		if execution.ComputeState.StateType == models.ExecutionStateCompleted {
			storageConfig, err := legacy.ToLegacyStorageSpec(execution.PublishedResult)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			results = append(results, model.PublishedResult{
				NodeID: execution.NodeID,
				Data:   storageConfig,
			})
		}
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
