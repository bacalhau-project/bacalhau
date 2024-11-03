//go:generate mockgen --source type.go --destination mocks.go --package bidstrategy
package bidstrategy

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type BidStrategyRequest struct {
	Job models.Job `json:"Job"`
}

type BidStrategyResponse struct {
	ShouldBid bool   `json:"ShouldBid"`
	Reason    string `json:"Reason"`
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

type ModerateJobRequest struct {
	ClientID string
	JobID    string
	Response BidStrategyResponse
}

func (mjr ModerateJobRequest) GetClientID() string {
	return mjr.ClientID
}
