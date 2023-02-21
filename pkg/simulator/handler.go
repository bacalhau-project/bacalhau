package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type RequestHandler struct {
	computeProxy   compute.Endpoint
	requesterProxy compute.Callback
	wallets        *walletsModel
	executionStore map[string]compute.ExecutionMetadata
}

func NewRequestHandler() *RequestHandler {
	return &RequestHandler{
		wallets:        newWalletsModel(),
		executionStore: make(map[string]compute.ExecutionMetadata),
	}
}

func (e *RequestHandler) SetComputeProxy(computeProxy compute.Endpoint) {
	e.computeProxy = computeProxy
}

func (e *RequestHandler) SetRequesterProxy(requesterProxy compute.Callback) {
	e.requesterProxy = requesterProxy
}

func (e *RequestHandler) AskForBid(ctx context.Context, request compute.AskForBidRequest) (compute.AskForBidResponse, error) {
	jobCreatedEvent := model.JobEvent{
		SourceNodeID: request.Job.Metadata.Requester.RequesterNodeID,
		JobID:        request.Job.Metadata.ID,
		EventName:    model.JobEventCreated,
		EventTime:    time.Now(),
	}
	err := e.wallets.addEvent(jobCreatedEvent)
	if err != nil {
		return compute.AskForBidResponse{}, err
	}

	response, err := e.computeProxy.AskForBid(ctx, request)
	if err != nil {
		return compute.AskForBidResponse{}, err
	}

	// only return the shard responses that the wallet did not reject
	toReturnShardResponses := make([]compute.AskForBidShardResponse, 0)
	for _, shardResponse := range response.ShardResponse {
		if shardResponse.Accepted {
			event := e.constructEvent(request.RoutingMetadata, shardResponse.ExecutionMetadata, model.JobEventBid)
			// we flip senders to mimic a bid was received instead of being asked
			event.SourceNodeID = request.RoutingMetadata.TargetPeerID
			err = e.wallets.addEvent(event)
			if err != nil {
				continue
			}
			e.executionStore[shardResponse.ExecutionMetadata.ExecutionID] = shardResponse.ExecutionMetadata
		}
		toReturnShardResponses = append(toReturnShardResponses, shardResponse)
	}
	response.ShardResponse = toReturnShardResponses

	return response, nil
}

func (e *RequestHandler) BidAccepted(ctx context.Context, request compute.BidAcceptedRequest) (compute.BidAcceptedResponse, error) {
	event, err := e.constructEventFromExecution(request.RoutingMetadata, request.ExecutionID, model.JobEventBidAccepted)
	if err != nil {
		return compute.BidAcceptedResponse{}, err
	}
	err = e.wallets.addEvent(event)
	if err != nil {
		return compute.BidAcceptedResponse{}, err
	}
	return e.computeProxy.BidAccepted(ctx, request)
}

func (e *RequestHandler) BidRejected(ctx context.Context, request compute.BidRejectedRequest) (compute.BidRejectedResponse, error) {
	event, err := e.constructEventFromExecution(request.RoutingMetadata, request.ExecutionID, model.JobEventBidRejected)
	if err != nil {
		return compute.BidRejectedResponse{}, err
	}
	err = e.wallets.addEvent(event)
	if err != nil {
		return compute.BidRejectedResponse{}, err
	}
	return e.computeProxy.BidRejected(ctx, request)
}

func (e *RequestHandler) ResultAccepted(
	ctx context.Context, request compute.ResultAcceptedRequest) (compute.ResultAcceptedResponse, error) {
	event, err := e.constructEventFromExecution(request.RoutingMetadata, request.ExecutionID, model.JobEventResultsAccepted)
	if err != nil {
		return compute.ResultAcceptedResponse{}, err
	}
	err = e.wallets.addEvent(event)
	if err != nil {
		return compute.ResultAcceptedResponse{}, err
	}
	return e.computeProxy.ResultAccepted(ctx, request)
}

func (e *RequestHandler) ResultRejected(
	ctx context.Context, request compute.ResultRejectedRequest) (compute.ResultRejectedResponse, error) {
	event, err := e.constructEventFromExecution(request.RoutingMetadata, request.ExecutionID, model.JobEventResultsRejected)
	if err != nil {
		return compute.ResultRejectedResponse{}, err
	}
	err = e.wallets.addEvent(event)
	if err != nil {
		return compute.ResultRejectedResponse{}, err
	}
	return e.computeProxy.ResultRejected(ctx, request)
}

func (e *RequestHandler) CancelExecution(
	ctx context.Context, request compute.CancelExecutionRequest) (compute.CancelExecutionResponse, error) {
	return e.computeProxy.CancelExecution(ctx, request)
}

func (e *RequestHandler) OnRunComplete(ctx context.Context, result compute.RunResult) {
	event, err := e.constructEventFromExecution(result.RoutingMetadata, result.ExecutionID, model.JobEventResultsProposed)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to construct event %s from execution %s", model.JobEventResultsProposed, result.ExecutionID)
	}
	event.VerificationProposal = result.ResultProposal
	event.RunOutput = result.RunCommandResult
	event.TargetNodeID = "" // requester node is never targeted

	err = e.wallets.addEvent(event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to add event %s from execution %s", model.JobEventResultsProposed, result.ExecutionID)
	}
	e.requesterProxy.OnRunComplete(ctx, result)
}

func (e *RequestHandler) OnPublishComplete(ctx context.Context, result compute.PublishResult) {
	event, err := e.constructEventFromExecution(result.RoutingMetadata, result.ExecutionID, model.JobEventResultsPublished)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to construct event %s from execution %s", model.JobEventResultsPublished, result.ExecutionID)
	}
	event.PublishedResult = result.PublishResult
	event.TargetNodeID = "" // requester node is never targeted

	err = e.wallets.addEvent(event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to add event %s from execution %s", model.JobEventResultsPublished, result.ExecutionID)
	}
	e.requesterProxy.OnPublishComplete(ctx, result)
}

func (e *RequestHandler) OnCancelComplete(ctx context.Context, result compute.CancelResult) {
	e.requesterProxy.OnCancelComplete(ctx, result)
}

func (e *RequestHandler) OnComputeFailure(ctx context.Context, result compute.ComputeError) {
	event, err := e.constructEventFromExecution(result.RoutingMetadata, result.ExecutionID, model.JobEventComputeError)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to construct event %s from execution %s", model.JobEventComputeError, result.ExecutionID)
	}
	event.Status = result.Error()
	event.TargetNodeID = "" // requester node is never targeted

	err = e.wallets.addEvent(event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to add event %s from execution %s", model.JobEventComputeError, result.ExecutionID)
	}
	e.requesterProxy.OnComputeFailure(ctx, result)
}

func (e *RequestHandler) constructEventFromExecution(
	routingMetadata compute.RoutingMetadata,
	executionID string,
	eventName model.JobEventType) (model.JobEvent, error) {
	executionMetadata, ok := e.executionStore[executionID]
	if !ok {
		return model.JobEvent{}, fmt.Errorf("execution id %s not found when trying to publish %s", executionID, eventName.String())
	}
	return e.constructEvent(routingMetadata, executionMetadata, eventName), nil
}

func (e *RequestHandler) constructEvent(
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

var _ compute.Endpoint = (*RequestHandler)(nil)
var _ compute.Callback = (*RequestHandler)(nil)
