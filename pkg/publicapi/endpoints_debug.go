package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/computenode"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

type debugResponse struct {
	AvailableComputeCapacity model.ResourceUsageData   `json:"AvailableComputeCapacity"`
	RequesterJobs            []requesternode.ActiveJob `json:"RequesterJobs"`
	ComputeJobs              []computenode.ActiveJob   `json:"ComputeJobs"`
}

// debug godoc
// @ID      apiServer/debug
// @Summary Returns debug information on what the current node is doing.
// @Tags    Health
// @Produce json
// @Success 200 {object} debugResponse
// @Failure 500 {object} string
// @Router  /debug [get]
func (apiServer *APIServer) debug(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/debug")
	defer span.End()

	responseObj := debugResponse{
		AvailableComputeCapacity: apiServer.ComputeNode.GetAvailableCapacity(ctx),
		RequesterJobs:            apiServer.Requester.GetActiveJobs(ctx),
		ComputeJobs:              apiServer.ComputeNode.GetActiveJobs(ctx),
	}

	res.WriteHeader(http.StatusOK)
	err := json.NewEncoder(res).Encode(responseObj)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
