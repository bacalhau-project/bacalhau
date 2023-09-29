//go:generate mockgen --source type.go --destination mocks.go --package bidstrategy
package bidstrategy

import (
	"context"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type BidStrategyRequest struct {
	NodeID   string
	Job      models.Job
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
	ShouldBidBasedOnUsage(ctx context.Context, request BidStrategyRequest, usage models.Resources) (BidStrategyResponse, error)
}

// the JSON data we send to http or exec probes
// TODO: can we just use the BidStrategyRequest struct?
type JobSelectionPolicyProbeData struct {
	NodeID   string     `json:"NodeID"`
	Job      models.Job `json:"Job"`
	Callback *url.URL   `json:"Callback"`
}

// Return JobSelectionPolicyProbeData for the given request
func GetJobSelectionPolicyProbeData(request BidStrategyRequest) JobSelectionPolicyProbeData {
	return JobSelectionPolicyProbeData(request)
}

type ModerateJobRequest struct {
	ClientID string
	JobID    string
	Response BidStrategyResponse
}

func (mjr ModerateJobRequest) GetClientID() string {
	return mjr.ClientID
}
