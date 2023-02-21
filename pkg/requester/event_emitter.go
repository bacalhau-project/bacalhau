package requester

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/model"
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
	ctx context.Context, job model.Job) {
	event := model.JobEvent{
		APIVersion:       job.APIVersion,
		ClientID:         job.Metadata.ClientID,
		Spec:             job.Spec,
		Deal:             job.Spec.Deal,
		JobExecutionPlan: job.Spec.ExecutionPlan,
		SourceNodeID:     job.Metadata.Requester.RequesterNodeID,
		JobID:            job.Metadata.ID,
		EventName:        model.JobEventCreated,
		EventTime:        time.Now(),
	}
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitBidReceived(
	ctx context.Context, request compute.AskForBidRequest, response compute.AskForBidShardResponse) {
	event := e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, model.JobEventBid)
	// we flip senders to mimic a bid was received instead of being asked
	event.SourceNodeID = request.RoutingMetadata.TargetPeerID
	event.TargetNodeID = "" // localdb don't assume a target node for events coming from compute nodes
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitBidAccepted(
	ctx context.Context, request compute.BidAcceptedRequest, response compute.BidAcceptedResponse) {
	event := e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, model.JobEventBidAccepted)
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitBidRejected(
	ctx context.Context, request compute.BidRejectedRequest, response compute.BidRejectedResponse) {
	event := e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, model.JobEventBidRejected)
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitResultAccepted(
	ctx context.Context, request compute.ResultAcceptedRequest, response compute.ResultAcceptedResponse) {
	event := e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, model.JobEventResultsAccepted)
	event.VerificationResult = model.VerificationResult{
		Complete: true,
		Result:   true,
	}
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitResultRejected(
	ctx context.Context, request compute.ResultRejectedRequest, response compute.ResultRejectedResponse) {
	event := e.constructEvent(request.RoutingMetadata, response.ExecutionMetadata, model.JobEventResultsRejected)
	event.VerificationResult = model.VerificationResult{
		Complete: true,
		Result:   false,
	}
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitRunComplete(ctx context.Context, response compute.RunResult) {
	event := e.constructEvent(response.RoutingMetadata, response.ExecutionMetadata, model.JobEventResultsProposed)
	event.VerificationProposal = response.ResultProposal
	event.RunOutput = response.RunCommandResult
	event.TargetNodeID = "" // localDB don't assume a target node for events coming from compute nodes
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitPublishComplete(ctx context.Context, response compute.PublishResult) {
	event := e.constructEvent(response.RoutingMetadata, response.ExecutionMetadata, model.JobEventResultsPublished)
	event.PublishedResult = response.PublishResult
	event.TargetNodeID = "" // localDB don't assume a target node for events coming from compute nodes
	e.EmitEventSilently(ctx, event)
}

func (e EventEmitter) EmitComputeFailure(ctx context.Context, response compute.ComputeError) {
	event := e.constructEvent(response.RoutingMetadata, response.ExecutionMetadata, model.JobEventComputeError)
	event.Status = response.Error()
	event.TargetNodeID = "" // localDB don't assume a target node for events coming from compute nodes
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
		ShardIndex:   executionMetadata.ShardIndex,
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
