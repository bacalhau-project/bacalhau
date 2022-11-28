package pubsub

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"go.opentelemetry.io/otel/trace"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type FrontendEventProxyParams struct {
	NodeID            string
	Frontend          frontend.Service
	JobStore          localdb.LocalDB
	ExecutionStore    store.ExecutionStore
	JobEventPublisher eventhandler.JobEventHandler
}

// FrontendEventProxy listens to events from GossipSub and forwards them to the frontend.
// This is a temporary solution that maintains backward compatibility with the current network until we fully switch
// to direct API calls for job orchestration.
type FrontendEventProxy struct {
	nodeID            string
	frontend          frontend.Service
	jobStore          localdb.LocalDB
	executionStore    store.ExecutionStore
	jobEventPublisher eventhandler.JobEventHandler
}

// NewFrontendEventProxy create a new FrontendEventProxy from FrontendEventProxyParams
func NewFrontendEventProxy(params FrontendEventProxyParams) *FrontendEventProxy {
	return &FrontendEventProxy{
		nodeID:            params.NodeID,
		frontend:          params.Frontend,
		jobStore:          params.JobStore,
		executionStore:    params.ExecutionStore,
		jobEventPublisher: params.JobEventPublisher,
	}
}

func (p FrontendEventProxy) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	ctx, span := p.newSpan(ctx, "HandleJobEvent")
	defer span.End()
	switch event.EventName {
	case model.JobEventCreated:
		return p.subscriptionEventCreated(ctx, event)
	case model.JobEventBidAccepted, model.JobEventBidRejected, model.JobEventResultsAccepted,
		model.JobEventResultsRejected, model.JobEventError:
		return p.triggerStateTransition(ctx, event)
	}
	return nil
}

func (p FrontendEventProxy) subscriptionEventCreated(ctx context.Context, event model.JobEvent) error {
	job, err := p.jobStore.GetJob(ctx, event.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("could not get job: %s - %s", event.JobID, err.Error())
		return nil
	}

	shardIndexes := []int{}
	for i := 0; i < job.ExecutionPlan.TotalShards; i++ {
		shardIndexes = append(shardIndexes, i)
	}
	request := frontend.AskForBidRequest{
		Job:          *job,
		ShardIndexes: shardIndexes,
	}
	response, err := p.frontend.AskForBid(ctx, request)
	if err != nil {
		return err
	}

	for _, shardResponse := range response.ShardResponse {
		if shardResponse.Accepted {
			notifyErr := p.processBidJob(ctx, shardResponse.ExecutionID)
			if notifyErr != nil {
				_, cancelError := p.frontend.CancelJob(ctx, frontend.CancelJobRequest{
					ExecutionID:   shardResponse.ExecutionID,
					Justification: "failed to notify bid",
				})
				if cancelError != nil {
					log.Ctx(ctx).Error().Msgf("error canceling execution after failing to notify bid: %s - %s",
						shardResponse.ExecutionID, cancelError.Error())
				}
			}
		}
	}
	return nil
}

func (p FrontendEventProxy) triggerStateTransition(ctx context.Context, event model.JobEvent) (err error) {
	// We ignore the event if it was sent to specific node that is not ours
	if event.TargetNodeID != "" && event.TargetNodeID != p.nodeID {
		return nil
	}
	shardID := model.GetShardID(event.JobID, event.ShardIndex)
	activeExecution, err := store.GetActiveExecution(ctx, p.executionStore, shardID)
	if err != nil {
		if !errors.As(err, &store.ErrExecutionNotFound{}) {
			return fmt.Errorf("error getting active execution: %w", err)
		} else {
			if event.TargetNodeID == p.nodeID {
				log.Ctx(ctx).Warn().Msgf(
					"received event targeted to this node for shard %s, but no execution state exists!", shardID)
			}
		}
		// we only care about events if it is related to a shard that we are executing
		return nil
	}

	switch event.EventName {
	case model.JobEventBidAccepted:
		request := frontend.BidAcceptedRequest{
			ExecutionID: activeExecution.ID,
		}
		_, err = p.frontend.BidAccepted(ctx, request)
	case model.JobEventBidRejected:
		request := frontend.BidRejectedRequest{
			ExecutionID: activeExecution.ID,
		}
		_, err = p.frontend.BidRejected(ctx, request)
	case model.JobEventResultsAccepted:
		request := frontend.ResultAcceptedRequest{
			ExecutionID: activeExecution.ID,
		}
		_, err = p.frontend.ResultAccepted(ctx, request)
	case model.JobEventResultsRejected:
		request := frontend.ResultRejectedRequest{
			ExecutionID: activeExecution.ID,
		}
		_, err = p.frontend.ResultRejected(ctx, request)
	case model.JobEventInvalidRequest, model.JobEventError:
		request := frontend.CancelJobRequest{
			ExecutionID:   activeExecution.ID,
			Justification: fmt.Sprintf("requester event %s triggered cancellation due to: %s", event.EventName, event.Status),
		}
		_, err = p.frontend.CancelJob(ctx, request)
	}
	return err
}

// Since some bid strategies might introduce a sleep delay, we need to make sure that the job is still
// accepting bids before we notify the requester that we have accepted the bid request.
func (p FrontendEventProxy) processBidJob(ctx context.Context, executionID string) error {
	execution, err := p.executionStore.GetExecution(ctx, executionID)
	if err != nil {
		return fmt.Errorf("error getting execution with id %s: %w", executionID, err)
	}

	jobState, err := p.jobStore.GetJobState(ctx, execution.Shard.Job.ID)
	if err != nil {
		return fmt.Errorf("error getting job state for job %s: %w", execution.Shard.Job.ID, err)
	}

	j, err := p.jobStore.GetJob(ctx, execution.Shard.Job.ID)
	if err != nil {
		return fmt.Errorf("error getting job with id %s: %w", execution.Shard.Job.ID, err)
	}

	hasShardReachedCapacity := jobutils.HasShardReachedCapacity(ctx, j, jobState, execution.Shard.Index)
	if hasShardReachedCapacity {
		_, cancelError := p.frontend.CancelJob(ctx, frontend.CancelJobRequest{
			ExecutionID:   executionID,
			Justification: "shard has reached capacity",
		})
		if cancelError != nil {
			log.Ctx(ctx).Error().Msgf("error canceling job: %s - %s", executionID, cancelError.Error())
		}
	}

	return p.jobEventPublisher.HandleJobEvent(ctx, p.constructEvent(ctx, execution, model.JobEventBid))
}

func (p FrontendEventProxy) constructEvent(ctx context.Context, execution store.Execution, eventName model.JobEventType) model.JobEvent {
	return model.JobEvent{
		SourceNodeID: p.nodeID,
		JobID:        execution.Shard.Job.ID,
		ShardIndex:   execution.Shard.Index,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
}

func (p FrontendEventProxy) newSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	// no need to set nodeID and jobID attributes, as they should already be set by the
	// chained event handler context provider
	return system.Span(ctx, "pkg/compute/proxy", name,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// compile-time check that FrontendEventProxy implements the eventhandler.JobEventHandler interface
var _ eventhandler.JobEventHandler = (*FrontendEventProxy)(nil)
