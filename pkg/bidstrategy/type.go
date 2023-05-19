package bidstrategy

import (
	"context"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type BidStrategyRequest struct {
	NodeID   string
	Job      model.Job
	Callback *url.URL
}

type BidStrategyResponse struct {
	ShouldBid  bool   `json:"shouldBid"`
	ShouldWait bool   `json:"shouldWait"`
	Reason     string `json:"reason"`
}

func NewShouldBidResponse() BidStrategyResponse {
	return BidStrategyResponse{
		ShouldBid: true,
	}
}

type BidStrategy interface {
	SemanticBidStrategy
	ResourceBidStrategy
}

type SemanticBidStrategy interface {
	ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error)
}

type ResourceBidStrategy interface {
	ShouldBidBasedOnUsage(ctx context.Context, request BidStrategyRequest, usage model.ResourceUsageData) (BidStrategyResponse, error)
}

// the JSON data we send to http or exec probes
// TODO: can we just use the BidStrategyRequest struct?
type JobSelectionPolicyProbeData struct {
	NodeID   string     `json:"node_id"`
	JobID    string     `json:"job_id"`
	Spec     model.Spec `json:"spec"`
	Callback *url.URL   `json:"callback"`
}

// Return JobSelectionPolicyProbeData for the given request
func GetJobSelectionPolicyProbeData(request BidStrategyRequest) JobSelectionPolicyProbeData {
	return JobSelectionPolicyProbeData{
		NodeID:   request.NodeID,
		JobID:    request.Job.Metadata.ID,
		Spec:     request.Job.Spec,
		Callback: request.Callback,
	}
}

type ModerateJobRequest struct {
	ClientID string
	JobID    string
	Response BidStrategyResponse
}

func (mjr ModerateJobRequest) GetClientID() string {
	return mjr.ClientID
}
