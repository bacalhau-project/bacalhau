package requesternode

// A job that is holding compute capacity, which can be in bidding or running state.
type ActiveJob struct {
	ShardID             string `json:"ShardID"`
	State               string `json:"State"`
	BiddingNodesCount   int    `json:"BiddingNodesCount"`
	CompletedNodesCount int    `json:"CompletedNodesCount"`
}
