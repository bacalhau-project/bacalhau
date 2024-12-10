package watchers

import (
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

type NCLMessageCreator struct {
}

func NewNCLMessageCreator() *NCLMessageCreator {
	return &NCLMessageCreator{}
}

func (d *NCLMessageCreator) CreateMessage(event watcher.Event) (*envelope.Message, error) {
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return nil, bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(nclDispatcherErrComponent)
	}

	execution := upsert.Current
	if execution.OrchestrationProtocol() != models.ProtocolNCLV1 {
		return nil, nil
	}

	// Prepare base response with common fields
	baseResponse := messages.BaseResponse{
		ExecutionID: execution.ID,
		JobID:       execution.JobID,
		JobType:     execution.Job.Type,
		Events:      upsert.Events}

	var message *envelope.Message
	// Create appropriate message based on execution state
	switch execution.ComputeState.StateType {
	case models.ExecutionStateAskForBidAccepted:
		log.Debug().Msgf("Accepting bid for execution %s", execution.ID)
		message = envelope.NewMessage(messages.BidResult{Accepted: true, BaseResponse: baseResponse}).
			WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType)
	case models.ExecutionStateAskForBidRejected:
		log.Debug().Msgf("Rejecting bid for execution %s", execution.ID)
		message = envelope.NewMessage(messages.BidResult{Accepted: false, BaseResponse: baseResponse}).
			WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType)
	case models.ExecutionStateCompleted:
		log.Debug().Msgf("Execution %s completed", execution.ID)
		message = envelope.NewMessage(messages.RunResult{
			BaseResponse:     baseResponse,
			PublishResult:    execution.PublishedResult,
			RunCommandResult: execution.RunOutput,
		}).WithMetadataValue(envelope.KeyMessageType, messages.RunResultMessageType)
	case models.ExecutionStateFailed:
		log.Debug().Msgf("Execution %s failed", execution.ID)
		message = envelope.NewMessage(messages.ComputeError{BaseResponse: baseResponse}).
			WithMetadataValue(envelope.KeyMessageType, messages.ComputeErrorMessageType)
	default:
		// No message created for other states
	}

	return message, nil
}

// compile-time check that NCLMessageCreator implements dispatcher.MessageCreator
var _ nclprotocol.MessageCreator = &NCLMessageCreator{}
