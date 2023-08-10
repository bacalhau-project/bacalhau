package orchestrator

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
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
	event := models.JobEvent{
		JobID:        job.Metadata.ID,
		SourceNodeID: job.Metadata.Requester.RequesterNodeID,
		EventName:    models.JobEventCreated,
		EventTime:    time.Now(),
	}
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitBidReceived(
	ctx context.Context, result compute.BidResult) {
	e.EmitEventSilently(ctx, e.constructEvent(result.RoutingMetadata, result.ExecutionMetadata, models.JobEventBid))
}

func (e EventEmitter) EmitBidAccepted(
	ctx context.Context, request compute.BidAcceptedRequest, response compute.BidAcceptedResponse) {
	e.EmitEventSilently(ctx, e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, models.JobEventBidAccepted))
}

func (e EventEmitter) EmitBidRejected(
	ctx context.Context, request compute.BidRejectedRequest, response compute.BidRejectedResponse) {
	e.EmitEventSilently(ctx, e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, models.JobEventBidRejected))
}

func (e EventEmitter) EmitRunComplete(ctx context.Context, response compute.RunResult) {
	e.EmitEventSilently(ctx, e.constructEvent(response.RoutingMetadata, response.ExecutionMetadata, models.JobEventResultsProposed))
}

func (e EventEmitter) EmitComputeFailure(ctx context.Context, executionID string, err error) {
	event := models.JobEvent{
		SourceNodeID: executionID.NodeID,
		JobID:        executionID.JobID,
		ExecutionID:  executionID.ExecutionID,
		EventName:    models.JobEventComputeError,
		Status:       err.Error(),
		EventTime:    time.Now(),
	}
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) constructEvent(
	routingMetadata compute.RoutingMetadata,
	executionMetadata compute.ExecutionMetadata,
	eventName models.JobEventType) models.JobEvent {
	return models.JobEvent{
		TargetNodeID: routingMetadata.TargetPeerID,
		SourceNodeID: routingMetadata.SourcePeerID,
		JobID:        executionMetadata.JobID,
		ExecutionID:  executionMetadata.ExecutionID,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
}

func (e EventEmitter) EmitEvent(ctx context.Context, event models.JobEvent) error {
	return e.eventConsumer.HandleJobEvent(ctx, event)
}

func (e EventEmitter) EmitEventSilently(ctx context.Context, event models.JobEvent) {
	err := e.EmitEvent(ctx, event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to emit event %+v", event)
	}
}
