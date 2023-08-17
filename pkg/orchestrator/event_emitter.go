package orchestrator

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/rs/zerolog/log"
)

// A quick workaround to publish job events locally as we still have some types that rely
// on job events to update their states (e.g. localdb) and to take actions (e.g. websockets and logging)
// TODO: create a strongly typed local event emitter similar to libp2p event bus, and update localdb directly from
//
//	requester instead of consuming events.
type EventEmitterParams struct {
	EventConsumer eventhandler.JobEventHandler
}

type EventEmitter struct {
	eventConsumer eventhandler.JobEventHandler
}

func NewEventEmitter(params EventEmitterParams) EventEmitter {
	return EventEmitter{
		eventConsumer: params.EventConsumer,
	}
}

func (e EventEmitter) EmitJobCreated(
	ctx context.Context, job models.Job) {
	event := model.JobEvent{
		JobID:        job.ID,
		SourceNodeID: job.Meta[models.MetaRequesterID],
		EventName:    model.JobEventCreated,
		EventTime:    time.Now(),
	}
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitBidReceived(
	ctx context.Context, result compute.BidResult) {
	e.EmitEventSilently(ctx, e.constructEvent(result.RoutingMetadata, result.ExecutionMetadata, model.JobEventBid))
}

func (e EventEmitter) EmitBidAccepted(
	ctx context.Context, request compute.BidAcceptedRequest, response compute.BidAcceptedResponse) {
	e.EmitEventSilently(ctx, e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, model.JobEventBidAccepted))
}

func (e EventEmitter) EmitBidRejected(
	ctx context.Context, request compute.BidRejectedRequest, response compute.BidRejectedResponse) {
	e.EmitEventSilently(ctx, e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, model.JobEventBidRejected))
}

func (e EventEmitter) EmitRunComplete(ctx context.Context, response compute.RunResult) {
	e.EmitEventSilently(ctx, e.constructEvent(response.RoutingMetadata, response.ExecutionMetadata, model.JobEventResultsProposed))
}

func (e EventEmitter) EmitComputeFailure(ctx context.Context, executionID string, err error) {
	event := model.JobEvent{
		ExecutionID: executionID,
		EventName:   model.JobEventComputeError,
		Status:      err.Error(),
		EventTime:   time.Now(),
	}
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) constructEvent(
	routingMetadata compute.RoutingMetadata,
	executionMetadata compute.ExecutionMetadata,
	eventName model.JobEventType) model.JobEvent {
	return model.JobEvent{
		TargetNodeID: routingMetadata.TargetPeerID,
		SourceNodeID: routingMetadata.SourcePeerID,
		JobID:        executionMetadata.JobID,
		ExecutionID:  executionMetadata.ExecutionID,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
}

func (e EventEmitter) EmitEvent(ctx context.Context, event model.JobEvent) error {
	return e.eventConsumer.HandleJobEvent(ctx, event)
}

func (e EventEmitter) EmitEventSilently(ctx context.Context, event model.JobEvent) {
	err := e.EmitEvent(ctx, event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to emit event %+v", event)
	}
}
