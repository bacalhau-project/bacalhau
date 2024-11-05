package watchers

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

type NCLDispatcher struct {
	publisher ncl.Publisher
	subjectFn func(nodeID string) string
	jobStore  jobstore.Store
}

type NCLDispatcherParams struct {
	Publisher ncl.Publisher
	SubjectFn func(nodeID string) string
	JobStore  jobstore.Store
}

// NewNCLDispatcher creates a new NCL protocol dispatcher
func NewNCLDispatcher(params NCLDispatcherParams) *NCLDispatcher {
	return &NCLDispatcher{
		publisher: params.Publisher,
		subjectFn: params.SubjectFn,
		jobStore:  params.JobStore,
	}
}

// HandleEvent implements watcher.EventHandler for NCL protocol
func (d *NCLDispatcher) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(nclDispatcherErrComponent)
	}

	// Skip if there's no state change
	if !upsert.HasStateChange() {
		return nil
	}

	message := d.createMessage(ctx, upsert)
	if message == nil {
		return nil
	}

	if err := d.publisher.Publish(ctx,
		ncl.NewPublishRequest(message).WithSubjectPrefix(d.subjectFn(upsert.Current.NodeID))); err != nil {
		return bacerrors.Wrap(err, "failed to publish message for execution %s", upsert.Current.ID).
			WithComponent(nclDispatcherErrComponent)
	}

	return nil
}

func (d *NCLDispatcher) createMessage(ctx context.Context, upsert models.ExecutionUpsert) *ncl.Message {
	transitions := newExecutionTransitions(upsert)

	switch {
	case transitions.shouldAskForPendingBid():
		return d.createAskForBidMessage(ctx, upsert)
	case transitions.shouldAskForDirectBid():
		return d.createAskForBidMessage(ctx, upsert)
	case transitions.shouldAcceptBid():
		return d.createBidAcceptedMessage(ctx, upsert)
	case transitions.shouldRejectBid():
		return d.createBidRejectedMessage(ctx, upsert)
	case transitions.shouldCancel():
		return d.createCancelMessage(ctx, upsert)
	}
	return nil
}

func (d *NCLDispatcher) createAskForBidMessage(ctx context.Context, upsert models.ExecutionUpsert) *ncl.Message {
	log.Ctx(ctx).Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Asking for bid")

	return ncl.NewMessage(messages.AskForBidRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		Execution:   upsert.Current,
	}).WithMetadataValue(ncl.KeyMessageType, messages.AskForBidMessageType)
}

func (d *NCLDispatcher) createBidAcceptedMessage(ctx context.Context, upsert models.ExecutionUpsert) *ncl.Message {
	log.Ctx(ctx).Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Accepting bid")

	return ncl.NewMessage(messages.BidAcceptedRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
		Accepted:    true,
	}).WithMetadataValue(ncl.KeyMessageType, messages.BidAcceptedMessageType)
}

func (d *NCLDispatcher) createBidRejectedMessage(ctx context.Context, upsert models.ExecutionUpsert) *ncl.Message {
	log.Ctx(ctx).Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Rejecting bid")

	return ncl.NewMessage(messages.BidRejectedRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
	}).WithMetadataValue(ncl.KeyMessageType, messages.BidRejectedMessageType)
}

func (d *NCLDispatcher) createCancelMessage(ctx context.Context, upsert models.ExecutionUpsert) *ncl.Message {
	log.Ctx(ctx).Debug().
		Str("nodeID", upsert.Current.NodeID).
		Str("executionID", upsert.Current.ID).
		Msg("Cancelling execution")

	// TODO: should not update the execution's observed state here. Should listen for compute events instead.
	// Mark execution as cancelled
	if err := d.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: upsert.Current.ID,
		NewValues: models.Execution{
			ComputeState: models.State[models.ExecutionStateType]{
				StateType: models.ExecutionStateCancelled,
			},
		},
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to mark execution %s as cancelled", upsert.Current.ID)
	}

	return ncl.NewMessage(messages.CancelExecutionRequest{
		BaseRequest: messages.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
	}).WithMetadataValue(ncl.KeyMessageType, messages.CancelExecutionMessageType)
}
