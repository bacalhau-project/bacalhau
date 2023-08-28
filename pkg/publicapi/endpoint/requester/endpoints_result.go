package requester

import (
	"encoding/json"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// results godoc
//
//	@ID						pkg/requester/publicapi/results
//	@Summary				Returns the results of the job-id specified in the body payload.
//	@Description.markdown	endpoints_results
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					StateRequest	body		apimodels.StateRequest	true	" "
//	@Success				200				{object}	apimodels.ResultsResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/api/v1/requester/results [post]
func (s *Endpoint) results(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var stateReq apimodels.StateRequest
	if err := json.NewDecoder(req.Body).Decode(&stateReq); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.Header().Set(apimodels.HTTPHeaderClientID, stateReq.ClientID)
	res.Header().Set(apimodels.HTTPHeaderJobID, stateReq.JobID)

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
	err = json.NewEncoder(res).Encode(apimodels.ResultsResponse{
		Results: results,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
