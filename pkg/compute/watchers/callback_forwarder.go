package watchers

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

// CallbackForwarder handles forwarding messages based on execution state changes
type CallbackForwarder struct {
	callback compute.Callback
}

// NewCallbackForwarder creates a new CallbackForwarder with the given callback
func NewCallbackForwarder(callback compute.Callback) *CallbackForwarder {
	return &CallbackForwarder{
		callback: callback,
	}
}

// HandleEvent processes a watcher event and publishes appropriate messages
func (h *CallbackForwarder) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(store.ExecutionUpsert)
	if !ok {
		return fmt.Errorf("failed to cast event object to Execution. Found type %T", event.Object)
	}

	execution := upsert.Current
	// Prepare base response with common fields
	routingMetadata := messages.RoutingMetadata{
		// the source of this response is the bidders nodeID.
		SourcePeerID: execution.NodeID,
		// the target of this response is the source of the request.
		TargetPeerID: execution.Job.Meta[models.MetaRequesterID],
	}
	executionMetadata := messages.NewExecutionMetadata(execution)
	updateEvent := models.Event{}
	if len(upsert.Events) > 0 {
		updateEvent = *upsert.Events[0]
	}

	// Create appropriate message based on execution state
	switch execution.ComputeState.StateType {
	case models.ExecutionStateAskForBidAccepted:
		log.Ctx(ctx).Debug().Msgf("Accepting bid for execution %s", execution.ID)
		h.callback.OnBidComplete(ctx, messages.BidResult{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Accepted:          true,
			Event:             updateEvent,
		})
	case models.ExecutionStateAskForBidRejected:
		log.Ctx(ctx).Debug().Msgf("Rejecting bid for execution %s", execution.ID)
		h.callback.OnBidComplete(ctx, messages.BidResult{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Accepted:          false,
			Event:             updateEvent,
		})
	case models.ExecutionStateBidAccepted:
		// Handle the case where bid is asked with pre-approval where compute state jumps to BidAccepted directly
		if upsert.Previous.ComputeState.StateType == models.ExecutionStateNew {
			log.Ctx(ctx).Debug().Msgf("Accepting and running execution %s", execution.ID)
			h.callback.OnBidComplete(ctx, messages.BidResult{
				RoutingMetadata:   routingMetadata,
				ExecutionMetadata: executionMetadata,
				Accepted:          true,
				Event:             updateEvent,
			})
		}
	case models.ExecutionStateCompleted:
		log.Ctx(ctx).Debug().Msgf("Execution %s completed", execution.ID)
		h.callback.OnRunComplete(ctx, messages.RunResult{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			PublishResult:     execution.PublishedResult,
			RunCommandResult:  execution.RunOutput,
		})
	case models.ExecutionStateFailed:
		log.Ctx(ctx).Debug().Msgf("Execution %s failed", execution.ID)
		h.callback.OnComputeFailure(ctx, messages.ComputeError{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Event:             updateEvent,
		})
	default:
		// No message created for other states
	}

	return nil
}
