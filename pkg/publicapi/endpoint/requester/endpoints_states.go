package requester

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/labstack/echo/v4"
)

// states godoc
//
//	@ID						pkg/requester/publicapi/states
//	@Summary				Returns the state of the job-id specified in the body payload.
//	@Description.markdown	endpoints_states
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					StateRequest	body		legacymodels.StateRequest	true	" "
//	@Success				200				{object}	legacymodels.StateResponse
//	@Failure				400				{object}	string
//	@Failure				500				{object}	string
//	@Router					/api/v1/requester/states [post]
func (s *Endpoint) states(c echo.Context) error {
	var stateReq legacymodels.StateRequest
	if err := c.Bind(&stateReq); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	c.Response().Header().Set(apimodels.HTTPHeaderClientID, stateReq.ClientID)
	c.Response().Header().Set(apimodels.HTTPHeaderJobID, stateReq.JobID)

	ctx := c.Request().Context()
	ctx = system.AddJobIDToBaggage(ctx, stateReq.JobID)

	js, err := legacy.GetJobState(ctx, s.jobStore, stateReq.JobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, legacymodels.StateResponse{State: js})
}
