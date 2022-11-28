package pubsub

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/backend"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type BackendCallbackParams struct {
	NodeID            string
	ExecutionStore    store.ExecutionStore
	JobEventPublisher eventhandler.JobEventHandler
}

// BackendCallback implements backend.Callback interface, which listens to backend events on job completion or
// cancellation, and forwards the events to GossipSub network.
// This is a temporary solution that maintains backward compatibility with the current network until we fully switch
// to direct API calls for job orchestration.
type BackendCallback struct {
	nodeID            string
	executionStore    store.ExecutionStore
	jobEventPublisher eventhandler.JobEventHandler
}

func NewBackendCallback(params BackendCallbackParams) *BackendCallback {
	return &BackendCallback{
		nodeID:            params.NodeID,
		executionStore:    params.ExecutionStore,
		jobEventPublisher: params.JobEventPublisher,
	}
}

func (p BackendCallback) OnRunSuccess(
	ctx context.Context,
	executionID string,
	result backend.RunResult,
) {
	ev, err := p.constructEvent(ctx, executionID, model.JobEventResultsProposed)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error constructing event: %s", err.Error())
		return
	}
	ev.VerificationProposal = result.ResultProposal
	ev.RunOutput = result.RunCommandResult
	p.publishEventSilently(ctx, ev)
}

func (p BackendCallback) OnRunFailure(
	ctx context.Context,
	executionID string,
	runError error,
) {
	ev, err2 := p.constructEvent(ctx, executionID, model.JobEventComputeError)
	if err2 != nil {
		log.Ctx(ctx).Error().Msgf("error constructing event: %s", err2.Error())
		return
	}
	ev.Status = runError.Error()
	p.publishEventSilently(ctx, ev)
}

func (p BackendCallback) OnPublishSuccess(
	ctx context.Context,
	executionID string,
	result backend.PublishResult,
) {
	ev, err := p.constructEvent(ctx, executionID, model.JobEventResultsPublished)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error constructing event: %s", err.Error())
		return
	}
	ev.PublishedResult = result.PublishResult
	p.publishEventSilently(ctx, ev)
}

func (p BackendCallback) OnPublishFailure(ctx context.Context, executionID string, err error) {
	log.Ctx(ctx).Error().Msgf("error publishing execution %s: %s", executionID, err)
}

func (p BackendCallback) OnCancelSuccess(ctx context.Context, executionID string, result backend.CancelResult) {
	log.Ctx(ctx).Info().Msgf("execution %s canceled successfully", executionID)
}

func (p BackendCallback) OnCancelFailure(ctx context.Context, executionID string, err error) {
	log.Ctx(ctx).Error().Msgf("error canceling execution %s: %s", executionID, err)
}

func (p BackendCallback) constructEvent(ctx context.Context, executionID string, eventName model.JobEventType) (model.JobEvent, error) {
	execution, err := p.executionStore.GetExecution(ctx, executionID)
	if err != nil {
		return model.JobEvent{}, err
	}
	return model.JobEvent{
		SourceNodeID: p.nodeID,
		JobID:        execution.Shard.Job.ID,
		ShardIndex:   execution.Shard.Index,
		EventName:    eventName,
		EventTime:    time.Now(),
	}, nil
}

func (p BackendCallback) publishEventSilently(ctx context.Context, ev model.JobEvent) {
	err := p.jobEventPublisher.HandleJobEvent(ctx, ev)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error publishing event: %s", err.Error())
	}
}

// compile-time check that BackendCallback implements the expected interfaces
var _ backend.Callback = (*BackendCallback)(nil)
