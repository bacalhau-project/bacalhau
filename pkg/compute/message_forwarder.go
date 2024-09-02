package compute

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// MessageForwarder handles forwarding messages based on execution state changes
type MessageForwarder struct {
	publisher ncl.Publisher
}

// NewForwarder creates a new MessageForwarder with the given publisher
func NewForwarder(publisher ncl.Publisher) *MessageForwarder {
	return &MessageForwarder{
		publisher,
	}
}

// HandleEvent processes a watcher event and publishes appropriate messages
func (h *MessageForwarder) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(store.ExecutionUpsert)
	if !ok {
		return fmt.Errorf("failed to cast event object to Execution. Found type %T", event.Object)
	}

	execution := upsert.Current
	// Prepare base response with common fields
	baseResponse := BaseResponse{
		ExecutionID: execution.ID,
		JobID:       execution.JobID,
		JobType:     execution.Job.Type,
		Events:      upsert.Events}

	var message *ncl.Message
	// Create appropriate message based on execution state
	switch execution.ComputeState.StateType {
	case models.ExecutionStateAskForBidAccepted:
		log.Ctx(ctx).Debug().Msgf("Accepting bid for execution %s", execution.ID)
		message = ncl.NewMessage(BidResult{Accepted: true, BaseResponse: baseResponse}).
			WithMetadataValue(ncl.KeyMessageType, BidResultMessageType)
	case models.ExecutionStateAskForBidRejected:
		log.Ctx(ctx).Debug().Msgf("Rejecting bid for execution %s", execution.ID)
		message = ncl.NewMessage(BidResult{Accepted: false, BaseResponse: baseResponse}).
			WithMetadataValue(ncl.KeyMessageType, BidResultMessageType)
	case models.ExecutionStateCompleted:
		log.Ctx(ctx).Debug().Msgf("Execution %s completed", execution.ID)
		message = ncl.NewMessage(RunResult{
			BaseResponse:     baseResponse,
			PublishResult:    execution.PublishedResult,
			RunCommandResult: execution.RunOutput,
		}).WithMetadataValue(ncl.KeyMessageType, RunResultMessageType)
	case models.ExecutionStateFailed:
		log.Ctx(ctx).Debug().Msgf("Execution %s failed", execution.ID)
		message = ncl.NewMessage(ComputeError{BaseResponse: baseResponse}).
			WithMetadataValue(ncl.KeyMessageType, ComputeErrorMessageType)
	default:
		// No message created for other states
	}

	// Publish the message if one was created
	if message != nil {
		return h.publisher.Publish(ctx, ncl.NewPublishRequest(message))
	}
	return nil
}
