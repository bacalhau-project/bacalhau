package semantic

import (
	"context"
	"hash/fnv"
	"math"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
)

// Decide whether we should even consider bidding on the job, early exit if
// we're not in the active set for this job, given the hash distances.
// (This is an optimization to avoid all nodes bidding on a job in large networks).

type DistanceDelayStrategyParams struct {
	NetworkSize int
}

// Compile-time check of interface implementation
var _ bidstrategy.SemanticBidStrategy = (*DistanceDelayStrategy)(nil)

type DistanceDelayStrategy struct {
	networkSize int
}

func NewDistanceDelayStrategy(params DistanceDelayStrategyParams) *DistanceDelayStrategy {
	return &DistanceDelayStrategy{networkSize: params.NetworkSize}
}

func (s DistanceDelayStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	jobNodeDistanceDelayMs, shouldRunJob := s.calculateJobNodeDistanceDelay(ctx, request)
	if !shouldRunJob {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    "Job to node hash distance too high",
		}, nil
	}

	if jobNodeDistanceDelayMs > 0 {
		log.Ctx(ctx).Debug().Msgf("Waiting %d ms before selecting job %s", jobNodeDistanceDelayMs, request.Job.Metadata.ID)
		time.Sleep(time.Millisecond * time.Duration(jobNodeDistanceDelayMs)) //nolint:gosec
	}

	return bidstrategy.NewShouldBidResponse(), nil
}

func (s DistanceDelayStrategy) calculateJobNodeDistanceDelay(ctx context.Context, request bidstrategy.BidStrategyRequest) (int, bool) {
	// Calculate how long to wait to bid on the job by using a circular hashing
	// style approach: Invent a metric for distance between node ID and job ID.
	// If the node and job ID happen to be close to eachother, such that we'd
	// expect that we are one of the N nodes "closest" to the job, bid
	// instantly. Beyond that, back off an amount "stepped" proportional to how
	// far we are from the job. This should evenly spread the work across the
	// network, and have the property of on average only concurrency many nodes
	// bidding on the job, and other nodes not bothering to bid because they
	// will already have seen bid/bidaccepted messages from the close nodes.
	// This will decrease overall network traffic, improving CPU and memory
	// usage in large clusters.
	nodeHash := hash(request.NodeID)
	jobHash := hash(request.Job.Metadata.ID)
	// Range: 0 through 4,294,967,295. (4 billion)
	distance := diff(nodeHash, jobHash)
	// scale distance per chunk by concurrency (so that many nodes bid on a job
	// with high concurrency). IOW, divide the space up into this many pieces.
	// If concurrency=3 and network size=3, there'll only be one piece and
	// everyone will bid. If concurrency=1 and network size=1 million, there
	// will be a million slices of the hash space.
	concurrency := max(1, request.Job.Spec.Deal.Concurrency, request.Job.Spec.Deal.MinBids)
	chunk := int((float32(concurrency) / float32(s.networkSize)) * 4294967295) //nolint:gomnd
	// wait 1 second per chunk distance. So, if we land in exactly the same
	// chunk, bid immediately. If we're one chunk away, wait a bit before
	// bidding. If we're very far away, wait a very long time.
	delay := (distance / chunk) * 1000 //nolint:gomnd
	log.Ctx(ctx).Trace().Msgf(
		"node/job %s/%s, %d/%d, dist=%d, chunk=%d, delay=%d",
		request.NodeID, request.Job.Metadata.ID, nodeHash, jobHash, distance, chunk, delay,
	)
	shouldRun := true
	// if delay is too high, just exit immediately.
	if delay > 1000 { //nolint:gomnd
		// drop the job on the floor, :-O
		shouldRun = false
		log.Ctx(ctx).Warn().Msgf(
			"dropped job: node/job %s/%s, %d/%d, dist=%d, chunk=%d, delay=%d",
			request.NodeID, request.Job.Metadata.ID, nodeHash, jobHash, distance, chunk, delay,
		)
	}
	return delay, shouldRun
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

func diff(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}

func max(vars ...int) int {
	res := math.MinInt

	for _, i := range vars {
		if res < i {
			res = i
		}
	}
	return res
}
