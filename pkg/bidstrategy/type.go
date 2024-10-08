//go:generate mockgen --source type.go --destination mocks.go --package bidstrategy
package bidstrategy

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type BidStrategyRequest struct {
	NodeID string
	Job    models.Job
}

type BidStrategyResponse struct {
	ShouldBid  bool   `json:"shouldBid"`
	ShouldWait bool   `json:"shouldWait"`
	Reason     string `json:"reason"`
}

const (
	reasonPrefix   string = "this node does "
	reasonNegation string = "not "
)

func FormatReason(success bool, jobRequiresNodeTo string, fmtArgs ...any) string {
	msg := reasonPrefix
	if !success {
		msg += reasonNegation
	}
	return fmt.Sprintf(msg+jobRequiresNodeTo, fmtArgs...)
}

func NewBidResponse(success bool, jobRequiresNodeTo string, fmtArgs ...any) BidStrategyResponse {
	return BidStrategyResponse{
		ShouldBid: success,
		Reason:    FormatReason(success, jobRequiresNodeTo, fmtArgs...),
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
	NodeID string     `json:"NodeID"`
	Job    models.Job `json:"Job"`
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
