package publicapi

import (
	"errors"
	"net/http"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
)

// approve godoc
//
//	@ID			apiServer/approver
//	@Summary	Approves a job to be run on this compute node.
//	@Produce	json
//	@Success	200	{object}	string
//	@Failure	400	{object}	string
//	@Failure	403	{object}	string
//	@Failure	500	{object}	string
//	@Router		/approve [get]
func (s *ComputeAPIServer) approve(res http.ResponseWriter, req *http.Request) {
	request, err := publicapi.UnmarshalSigned[bidstrategy.ModerateJobRequest](req.Context(), req.Body)
	if err != nil {
		publicapi.HTTPError(req.Context(), res, err, http.StatusBadRequest)
		return
	}

	approvingClient := os.Getenv("BACALHAU_JOB_APPROVER")
	if request.ClientID != approvingClient {
		err := errors.New("approval submitted by unknown client")
		publicapi.HTTPError(req.Context(), res, err, http.StatusUnauthorized)
		return
	}

	executions, err := s.store.GetExecutions(req.Context(), request.JobID)
	if err != nil {
		publicapi.HTTPError(req.Context(), res, err, http.StatusInternalServerError)
		return
	}

	for _, execution := range executions {
		go s.bidder.ReturnBidResult(req.Context(), execution, &request.Response)
	}
}
