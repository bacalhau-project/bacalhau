package compute

import (
	"errors"
	"net/http"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/signatures"
	"github.com/labstack/echo/v4"
)

// approve godoc
//
//	@ID			apiServer/approver
//	@Summary	Approves a job to be run on this compute node.
//	@Tags		Compute Node
//	@Produce	json
//	@Success	200	{object}	string
//	@Failure	400	{object}	string
//	@Failure	403	{object}	string
//	@Failure	500	{object}	string
//	@Router		/api/v1/compute/approve [get]
func (s *Endpoint) approve(c echo.Context) error {
	request, err := signatures.UnmarshalSigned[bidstrategy.ModerateJobRequest](c.Request().Context(), c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	approvingClient := os.Getenv("BACALHAU_JOB_APPROVER")
	if request.ClientID != approvingClient {
		return echo.NewHTTPError(http.StatusUnauthorized, errors.New("approval submitted by unknown client"))
	}

	executions, err := s.store.GetExecutions(c.Request().Context(), request.JobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	for _, execution := range executions {
		go s.bidder.ReturnBidResult(c.Request().Context(), execution, &request.Response)
	}

	return c.JSON(http.StatusOK, "Job approved.")
}
