package compute

import (
	"context"
	"reflect"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

type MessageHandler struct {
	executionStore store.ExecutionStore
}

func NewMessageHandler(executionStore store.ExecutionStore) *MessageHandler {
	return &MessageHandler{
		executionStore: executionStore,
	}
}

func (m *MessageHandler) ShouldProcess(ctx context.Context, message *ncl.Message) bool {
	return message.Metadata.Get(ncl.KeyMessageType) == messages.AskForBidMessageType ||
		message.Metadata.Get(ncl.KeyMessageType) == messages.BidAcceptedMessageType ||
		message.Metadata.Get(ncl.KeyMessageType) == messages.BidRejectedMessageType ||
		message.Metadata.Get(ncl.KeyMessageType) == messages.CancelExecutionMessageType
}

// HandleMessage handles incoming messages
// TODO: handle messages arriving out of order gracefully
func (m *MessageHandler) HandleMessage(ctx context.Context, message *ncl.Message) error {
	switch message.Metadata.Get(ncl.KeyMessageType) {
	case messages.AskForBidMessageType:
		return m.handleAskForBid(ctx, message)
	case messages.BidAcceptedMessageType:
		return m.handleBidAccepted(ctx, message)
	case messages.BidRejectedMessageType:
		return m.handleBidRejected(ctx, message)
	case messages.CancelExecutionMessageType:
		return m.handleCancel(ctx, message)
	default:
		return nil
	}
}

func (m *MessageHandler) handleAskForBid(ctx context.Context, message *ncl.Message) error {
	request, ok := message.Payload.(*messages.AskForBidRequest)
	if !ok {
		return ncl.NewErrUnexpectedPayloadType("AskForBidRequest", reflect.TypeOf(message.Payload).String())
	}

	// Set the protocol version in the job meta
	execution := request.Execution
	if execution.Job == nil {
		return bacerrors.New("job is missing in the execution").WithComponent(messageHandlerErrorComponent)
	}
	execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolNCLV1.String()

	// Create the execution
	return m.executionStore.CreateExecution(ctx, *request.Execution)
}

func (m *MessageHandler) handleBidAccepted(ctx context.Context, message *ncl.Message) error {
	request, ok := message.Payload.(*messages.BidAcceptedRequest)
	if !ok {
		return ncl.NewErrUnexpectedPayloadType("BidAcceptedRequest", reflect.TypeOf(message.Payload).String())
	}

	log.Ctx(ctx).Debug().Msgf("bid accepted %s", request.ExecutionID)
	return m.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: request.ExecutionID,
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				models.ExecutionStateNew, models.ExecutionStateAskForBidAccepted},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted),
		},
	})
}

func (m *MessageHandler) handleBidRejected(ctx context.Context, message *ncl.Message) error {
	request, ok := message.Payload.(*messages.BidRejectedRequest)
	if !ok {
		return ncl.NewErrUnexpectedPayloadType("BidRejectedRequest", reflect.TypeOf(message.Payload).String())
	}

	log.Ctx(ctx).Debug().Msgf("bid rejected for %s due to %s", request.ExecutionID, request.Message())
	return m.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: request.ExecutionID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateBidRejected).WithMessage(request.Message()),
		},
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				models.ExecutionStateNew, models.ExecutionStateAskForBidAccepted},
		},
		Events: request.Events,
	})
}

func (m *MessageHandler) handleCancel(ctx context.Context, message *ncl.Message) error {
	request, ok := message.Payload.(*messages.CancelExecutionRequest)
	if !ok {
		return ncl.NewErrUnexpectedPayloadType("CancelExecutionRequest", reflect.TypeOf(message.Payload).String())
	}

	log.Ctx(ctx).Debug().Msgf("canceling execution %s due to %s", request.ExecutionID, request.Message())
	return m.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: request.ExecutionID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateCancelled).WithMessage(request.Message()),
		},
		Events: request.Events,
	})
}

// compile time check for the interface
var _ ncl.MessageHandler = &MessageHandler{}
