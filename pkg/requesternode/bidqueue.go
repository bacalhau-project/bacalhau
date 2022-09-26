package requesternode

import (
	"context"
	"math/rand"

	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type bidQueueResult struct {
	nodeID   string
	accepted bool
}

func filterLocalEvents(
	ctx context.Context,
	events []model.JobLocalEvent,
	eventType model.JobLocalEventType,
) []model.JobLocalEvent {
	filteredEvents := []model.JobLocalEvent{}
	for _, event := range events {
		if event.EventName == eventType {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
}

// let's see how many bids we have already accepted
// it's important this comes from "local events"
// otherwise we are in a race with the network and could
// end up accepting many more bids than our concurrency
func getLocalShardEvents(
	ctx context.Context,
	controller *controller.Controller,
	jobID string,
	shardIndex int,
) ([]model.JobLocalEvent, error) {
	localEvents, err := controller.GetJobLocalEvents(ctx, jobID)
	if err != nil {
		return nil, err
	}
	shardLocalEvents := []model.JobLocalEvent{}
	for _, localEvent := range localEvents {
		if localEvent.ShardIndex == shardIndex {
			shardLocalEvents = append(shardLocalEvents, localEvent)
		}
	}
	return shardLocalEvents, nil
}

// these are the bids we have heard about
func getGlobalShardBidEvents(
	ctx context.Context,
	controller *controller.Controller,
	jobID string,
	shardIndex int,
) ([]model.JobEvent, error) {
	globalEvents, err := controller.GetJobEvents(ctx, jobID)
	if err != nil {
		return nil, err
	}
	shardGlobalEvents := []model.JobEvent{}
	for _, globalEvent := range globalEvents { //nolint:gocritic
		if globalEvent.EventName == model.JobEventBid && globalEvent.ShardIndex == shardIndex {
			shardGlobalEvents = append(shardGlobalEvents, globalEvent)
		}
	}
	return shardGlobalEvents, nil
}

// filter the global bid events down to ones
// we've not responded to yet
// all these lists of events are already filtered down to the shard level
func getCandidateBids(
	ctx context.Context,
	bidEvents []model.JobEvent,
	acceptedEvents []model.JobLocalEvent,
	rejectedEvents []model.JobLocalEvent,
) []model.JobEvent {
	allRespondedEvents := []model.JobLocalEvent{}
	allRespondedEvents = append(allRespondedEvents, acceptedEvents...)
	allRespondedEvents = append(allRespondedEvents, rejectedEvents...)

	hostsResponded := map[string]bool{}

	// loop over existing responses and build a map of hosts we've already responded to
	for _, respondedEvent := range allRespondedEvents { //nolint:gocritic
		hostsResponded[respondedEvent.TargetNodeID] = true
	}

	candidateBids := []model.JobEvent{}

	// loop over bidEvents and filter out the ones we've already responded to
	for _, bidEvent := range bidEvents { //nolint:gocritic
		if _, ok := hostsResponded[bidEvent.SourceNodeID]; !ok {
			candidateBids = append(candidateBids, bidEvent)
		}
	}

	// randomize the candidateBids slice before returning it
	rand.Shuffle(len(candidateBids), func(i, j int) {
		candidateBids[i], candidateBids[j] = candidateBids[j], candidateBids[i]
	})

	return candidateBids
}

// we just heard a compute node bid on a job we are looking after
// we need to check min bids and see what bids we have already
// accepted - we return two lists, "bids to accept" and "bids to reject"
// both lists can be empty in the case that we've not heard enough bids yet
func processIncomingBid(
	ctx context.Context,
	controller *controller.Controller,
	j *model.Job,
	jobEvent model.JobEvent,
) ([]bidQueueResult, error) {
	// global bid events we've heard for this shard
	bidsHeard, err := getGlobalShardBidEvents(ctx, controller, j.ID, jobEvent.ShardIndex)
	if err != nil {
		return nil, err
	}

	// all local events for this shard
	localEvents, err := getLocalShardEvents(ctx, controller, j.ID, jobEvent.ShardIndex)
	if err != nil {
		return nil, err
	}
	// process into local accepted and rejected
	bidsAccepted := filterLocalEvents(ctx, localEvents, model.JobLocalEventBidAccepted)
	bidsRejected := filterLocalEvents(ctx, localEvents, model.JobLocalEventBidRejected)
	// from the global bids we've heard, filter out the ones we've already responded to
	candidateBids := getCandidateBids(ctx, bidsHeard, bidsAccepted, bidsRejected)

	results := []bidQueueResult{}
	minBids := j.Deal.MinBids
	concurrency := j.Deal.Concurrency

	// main control switch
	if len(bidsHeard) < minBids {
		// if we have not heard enough bids yet - we don't do anything until we have heard enough
		return results, nil
	} else if len(bidsHeard) == minBids {
		// we've reached our threshold of when we can start accepting bids
		// first let's randomize the list of bids
		// then pick the first concurrency number of them to accept and reject the rest
		// if min bids < concurrency then we accept them all
		bidsToAcceptCount := len(candidateBids)
		if bidsToAcceptCount > concurrency {
			bidsToAcceptCount = concurrency
		}

		for i := 0; i < len(candidateBids); i++ {
			candidateBid := candidateBids[i]
			results = append(results, bidQueueResult{
				nodeID:   candidateBid.SourceNodeID,
				accepted: i < bidsToAcceptCount,
			})
		}

		return results, nil
	} else {
		// we've just heard of a bid and we've already exceeded our min bids threshold
		// so we are checking concurrency against accepeted bids
		results = []bidQueueResult{
			{
				nodeID:   jobEvent.SourceNodeID,
				accepted: len(bidsAccepted) < concurrency,
			},
		}
		return results, nil
	}
}
