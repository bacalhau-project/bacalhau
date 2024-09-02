package orchestrator

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type MessageForwarderSubjectFn func(nodeID string) string

type MessageForwarder struct {
	publisher ncl.Publisher
	subjectFn MessageForwarderSubjectFn
}

func NewExecutionForwarder(publisher ncl.Publisher, subjectFn MessageForwarderSubjectFn) *MessageForwarder {
	return &MessageForwarder{
		publisher: publisher,
		subjectFn: subjectFn,
	}
}

func (h *MessageForwarder) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(jobstore.ExecutionUpsert)
	if !ok {
		return fmt.Errorf("failed to cast event object to jobstore.ExecutionUpsert. Found type %T", event.Object)
	}

	execution := upsert.Current
	desiredState := execution.DesiredState.StateType
	observedState := execution.ComputeState.StateType
	log.Trace().Msgf("Handling event for execution %s. Desired state: %s, observed state: %s",
		execution, desiredState.String(), observedState.String())

	var message *ncl.Message
	switch desiredState {
	case models.ExecutionDesiredStatePending:
		if observedState == models.ExecutionStateNew {
			message = h.askForBid(message, upsert)
		}
	case models.ExecutionDesiredStateRunning:
		if observedState == models.ExecutionStateNew {
			h.askForBid(message, upsert)
		}
		if observedState == models.ExecutionStateAskForBidAccepted {
			message = h.bidAccepted(message, upsert)
		}
	case models.ExecutionDesiredStateStopped:
		if observedState == models.ExecutionStateAskForBidAccepted {
			message = h.bidRejected(message, upsert)
		} else if !execution.IsTerminalComputeState() {
			message = h.cancelExecution(message, upsert)
		}
	}

	if message != nil {
		return h.publisher.Publish(ctx,
			ncl.NewPublishRequest(message).WithSubjectPrefix(h.subjectFn(execution.NodeID)),
		)
	}
	return nil
}

func (h *MessageForwarder) askForBid(message *ncl.Message, upsert jobstore.ExecutionUpsert) *ncl.Message {
	log.Debug().Msgf("Asking node %s for bid for execution %s", upsert.Current.NodeID, upsert.Current.ID)
	message = ncl.NewMessage(compute.AskForBidRequest{
		BaseRequest: compute.BaseRequest{Events: upsert.Events},
		Execution:   upsert.Current,
	}).WithMetadataValue(ncl.KeyMessageType, compute.AskForBidMessageType)
	return message
}

func (h *MessageForwarder) bidAccepted(message *ncl.Message, upsert jobstore.ExecutionUpsert) *ncl.Message {
	log.Debug().Msgf("Bid accepted for execution %s from node %s", upsert.Current.ID, upsert.Current.NodeID)
	message = ncl.NewMessage(compute.BidAcceptedRequest{
		BaseRequest: compute.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
		Accepted:    true,
	}).WithMetadataValue(ncl.KeyMessageType, compute.BidAcceptedMessageType)
	return message
}

func (h *MessageForwarder) bidRejected(message *ncl.Message, upsert jobstore.ExecutionUpsert) *ncl.Message {
	log.Debug().Msgf("Bid rejected for execution %s from node %s", upsert.Current.ID, upsert.Current.NodeID)
	message = ncl.NewMessage(compute.BidRejectedRequest{
		BaseRequest: compute.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
	}).WithMetadataValue(ncl.KeyMessageType, compute.BidRejectedMessageType)
	return message
}

func (h *MessageForwarder) cancelExecution(message *ncl.Message, upsert jobstore.ExecutionUpsert) *ncl.Message {
	log.Debug().Msgf("Cancelling execution %s from node %s due to %s",
		upsert.Current.ID, upsert.Current.NodeID, upsert.Current.DesiredState.Message)
	message = ncl.NewMessage(compute.CancelExecutionRequest{
		BaseRequest: compute.BaseRequest{Events: upsert.Events},
		ExecutionID: upsert.Current.ID,
	}).WithMetadataValue(ncl.KeyMessageType, compute.CancelExecutionMessageType)
	return message
}
