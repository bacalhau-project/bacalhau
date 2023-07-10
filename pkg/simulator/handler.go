package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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

	return e.computeProxy.AskForBid(ctx, request)
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

func (e *RequestHandler) CancelExecution(
	ctx context.Context, request compute.CancelExecutionRequest) (compute.CancelExecutionResponse, error) {
	return e.computeProxy.CancelExecution(ctx, request)
}

func (e *RequestHandler) ExecutionLogs(
	ctx context.Context, request compute.ExecutionLogsRequest) (compute.ExecutionLogsResponse, error) {
	return e.computeProxy.ExecutionLogs(ctx, request)
}

func (e *RequestHandler) OnBidComplete(ctx context.Context, result compute.BidResult) {
	e.executionStore[result.ExecutionMetadata.ExecutionID] = result.ExecutionMetadata
	if result.Accepted {
		event, err := e.constructEventFromExecution(result.RoutingMetadata, result.ExecutionID, model.JobEventBid)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to construct event %s from execution %s", model.JobEventBid, result.ExecutionID)
		}
		err = e.wallets.addEvent(event)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to add event %s from execution %s", model.JobEventBid, result.ExecutionID)
		}
	}
	e.requesterProxy.OnBidComplete(ctx, result)
}

func (e *RequestHandler) OnRunComplete(ctx context.Context, result compute.RunResult) {
	event, err := e.constructEventFromExecution(result.RoutingMetadata, result.ExecutionID, model.JobEventResultsProposed)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to construct event %s from execution %s", model.JobEventResultsProposed, result.ExecutionID)
	}
	err = e.wallets.addEvent(event)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to add event %s from execution %s", model.JobEventResultsProposed, result.ExecutionID)
	}
	e.requesterProxy.OnRunComplete(ctx, result)
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
		ExecutionID:  executionMetadata.ExecutionID,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
}

var _ compute.Endpoint = (*RequestHandler)(nil)
var _ compute.Callback = (*RequestHandler)(nil)
