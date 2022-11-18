package computenode

import "github.com/filecoin-project/bacalhau/pkg/model"

// A job that is holding compute capacity, which can be in bidding or running state.
type ActiveJob struct {
	ShardID              string                  `json:"ShardID"`
	State                string                  `json:"State"`
	CapacityRequirements model.ResourceUsageData `json:"CapacityRequirements"`
}
