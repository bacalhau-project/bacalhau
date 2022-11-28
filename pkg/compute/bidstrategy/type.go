package bidstrategy

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type BidStrategyRequest struct {
	NodeID string
	Job    model.Job
}

type BidStrategyResponse struct {
	ShouldBid bool
	Reason    string
}

func newShouldBidResponse() BidStrategyResponse {
	return BidStrategyResponse{
		ShouldBid: true,
	}
}

type BidStrategy interface {
	ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error)
	ShouldBidBasedOnUsage(ctx context.Context, request BidStrategyRequest, resourceUsage model.ResourceUsageData) (BidStrategyResponse, error)
}

// the JSON data we send to http or exec probes
// TODO: can we just use the BidStrategyRequest struct?
type JobSelectionPolicyProbeData struct {
	NodeID        string                 `json:"node_id"`
	JobID         string                 `json:"job_id"`
	Spec          model.Spec             `json:"spec"`
	ExecutionPlan model.JobExecutionPlan `json:"execution_plan"`
}

// Return JobSelectionPolicyProbeData for the given request
func getJobSelectionPolicyProbeData(request BidStrategyRequest) JobSelectionPolicyProbeData {
	return JobSelectionPolicyProbeData{
		NodeID:        request.NodeID,
		JobID:         request.Job.ID,
		Spec:          request.Job.Spec,
		ExecutionPlan: request.Job.ExecutionPlan,
	}
}
