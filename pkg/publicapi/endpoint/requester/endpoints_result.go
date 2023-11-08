package requester

import (
	"net/http"

	"github.com/labstack/echo/v4"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// results godoc
//
//	@ID						pkg/requester/publicapi/results
//	@Summary				Returns the results of the job-id specified in the body payload.
//	@Description.markdown	endpoints_results
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					StateRequest	body		legacymodels.StateRequest	true	" "
//	@Success				200				{object}	legacymodels.ResultsResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/api/v1/requester/results [post]
func (s *Endpoint) results(c echo.Context) error {
	var stateReq legacymodels.StateRequest
	if err := c.Bind(&stateReq); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	c.Response().Header().Set(apimodels.HTTPHeaderClientID, stateReq.ClientID)
	c.Response().Header().Set(apimodels.HTTPHeaderJobID, stateReq.JobID)

	ctx := c.Request().Context()
	ctx = system.AddJobIDToBaggage(ctx, stateReq.JobID)
	system.AddJobIDFromBaggageToSpan(ctx, oteltrace.SpanFromContext(ctx))

	executions, err := s.jobStore.GetExecutions(ctx, stateReq.JobID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	results := make([]model.PublishedResult, 0)
	for _, execution := range executions {
		if execution.ComputeState.StateType == models.ExecutionStateCompleted {
			// TODO this is making adding different publishers really hard.
			storageConfig, err := legacy.ToLegacyStorageSpec(execution.PublishedResult)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, err.Error())
			}
			results = append(results, model.PublishedResult{
				NodeID: execution.NodeID,
				Data:   storageConfig,
			})
		}
	}

	return c.JSON(http.StatusOK, legacymodels.ResultsResponse{Results: results})
}
