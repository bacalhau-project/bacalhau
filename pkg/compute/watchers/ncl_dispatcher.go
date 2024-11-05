package watchers

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

// NCLDispatcher handles forwarding messages based on execution state changes
type NCLDispatcher struct {
	publisher ncl.Publisher
}

// NewNCLDispatcher creates a new NCLDispatcher with the given publisher
func NewNCLDispatcher(publisher ncl.Publisher) *NCLDispatcher {
	return &NCLDispatcher{
		publisher,
	}
}

// HandleEvent processes a watcher event and publishes appropriate messages
func (d *NCLDispatcher) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(nclDispatcherErrComponent)
	}

	execution := upsert.Current

	// Prepare base response with common fields
	baseResponse := messages.BaseResponse{
		ExecutionID: execution.ID,
		JobID:       execution.JobID,
		JobType:     execution.Job.Type,
		Events:      upsert.Events}

	var message *ncl.Message
	// Create appropriate message based on execution state
	switch execution.ComputeState.StateType {
	case models.ExecutionStateAskForBidAccepted:
		log.Ctx(ctx).Debug().Msgf("Accepting bid for execution %s", execution.ID)
		message = ncl.NewMessage(messages.BidResult{Accepted: true, BaseResponse: baseResponse}).
			WithMetadataValue(ncl.KeyMessageType, messages.BidResultMessageType)
	case models.ExecutionStateAskForBidRejected:
		log.Ctx(ctx).Debug().Msgf("Rejecting bid for execution %s", execution.ID)
		message = ncl.NewMessage(messages.BidResult{Accepted: false, BaseResponse: baseResponse}).
			WithMetadataValue(ncl.KeyMessageType, messages.BidResultMessageType)
	case models.ExecutionStateCompleted:
		log.Ctx(ctx).Debug().Msgf("Execution %s completed", execution.ID)
		message = ncl.NewMessage(messages.RunResult{
			BaseResponse:     baseResponse,
			PublishResult:    execution.PublishedResult,
			RunCommandResult: execution.RunOutput,
		}).WithMetadataValue(ncl.KeyMessageType, messages.RunResultMessageType)
	case models.ExecutionStateFailed:
		log.Ctx(ctx).Debug().Msgf("Execution %s failed", execution.ID)
		message = ncl.NewMessage(messages.ComputeError{BaseResponse: baseResponse}).
			WithMetadataValue(ncl.KeyMessageType, messages.ComputeErrorMessageType)
	default:
		// No message created for other states
	}

	// Publish the message if one was created
	if message != nil {
		return d.publisher.Publish(ctx, ncl.NewPublishRequest(message))
	}
	return nil
}
