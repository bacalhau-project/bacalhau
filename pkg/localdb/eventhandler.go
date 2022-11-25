package localdb

import (
	"context"

	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

// An event handler that listens to both job and local events, and updates the LocalDB instance accordingly
type LocalDBEventHandler struct {
	localDB LocalDB
}

func NewLocalDBEventHandler(localDB LocalDB) *LocalDBEventHandler {
	return &LocalDBEventHandler{
		localDB: localDB,
	}
}

func (h *LocalDBEventHandler) HandleLocalEvent(ctx context.Context, event model.JobLocalEvent) error {
	switch event.EventName {
	case model.JobLocalEventBidAccepted,
		model.JobLocalEventBidRejected,
		model.JobLocalEventVerified,
		model.JobLocalEventSelected,
		model.JobLocalEventBid:
		return h.localDB.AddLocalEvent(ctx, event.JobID, event)
	}
	return nil
}

func (h *LocalDBEventHandler) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	var err error
	switch event.EventName {
	case model.JobEventCreated:
		j := jobutils.ConstructJobFromEvent(event)
		err = h.localDB.AddJob(ctx, j)
	case model.JobEventDealUpdated:
		err = h.localDB.UpdateJobDeal(ctx, event.JobID, event.Deal)
	}

	if err != nil {
		return err
	}
	log.Debug().Msgf("JobID = %s", event.JobID)

	err = h.localDB.AddEvent(ctx, event.JobID, event)
	if err != nil {
		return err
	}

	executionState := model.GetStateFromEvent(event.EventName)

	// in most cases - the source node is the id of the state
	// we are updating - there are a few events where the target node id
	// overrides this (e.g. BidAccepted)
	useNodeID := event.SourceNodeID
	if event.TargetNodeID != "" {
		useNodeID = event.TargetNodeID
	}

	if model.IsValidJobState(executionState) {
		// update the state for this job shard
		err = h.localDB.UpdateShardState(
			ctx,
			event.JobID,
			useNodeID,
			event.ShardIndex,
			model.JobShardState{
				NodeID:               useNodeID,
				ShardIndex:           event.ShardIndex,
				State:                executionState,
				Status:               event.Status,
				VerificationProposal: event.VerificationProposal,
				VerificationResult:   event.VerificationResult,
				PublishedResult:      event.PublishedResult,
				RunOutput:            event.RunOutput,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}
