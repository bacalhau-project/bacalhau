package requester

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const OverAskForBidsFactor = 3 // ask up to 3 times the desired number of bids

type SchedulerParams struct {
	ID                                 string
	Host                               host.Host
	JobStore                           localdb.LocalDB
	NodeDiscoverer                     NodeDiscoverer
	NodeRanker                         NodeRanker
	ComputeEndpoint                    compute.Endpoint
	Verifiers                          verifier.VerifierProvider
	StorageProviders                   storage.StorageProvider
	EventEmitter                       EventEmitter
	JobNegotiationTimeout              time.Duration
	StateManagerBackgroundTaskInterval time.Duration
}

type Scheduler struct {
	id                string
	host              host.Host
	jobStore          localdb.LocalDB
	nodeDiscoverer    NodeDiscoverer
	nodeRanker        NodeRanker
	computeService    compute.Endpoint
	verifiers         verifier.VerifierProvider
	storageProviders  storage.StorageProvider
	eventEmitter      EventEmitter
	shardStateManager *shardStateMachineManager
}

func NewScheduler(ctx context.Context, cm *system.CleanupManager, params SchedulerParams) *Scheduler {
	return &Scheduler{
		id:               params.ID,
		host:             params.Host,
		jobStore:         params.JobStore,
		nodeDiscoverer:   params.NodeDiscoverer,
		nodeRanker:       params.NodeRanker,
		computeService:   params.ComputeEndpoint,
		verifiers:        params.Verifiers,
		storageProviders: params.StorageProviders,
		eventEmitter:     params.EventEmitter,
		shardStateManager: newShardStateMachineManager(
			ctx, cm, params.JobNegotiationTimeout, params.StateManagerBackgroundTaskInterval),
	}
}

func (s *Scheduler) StartJob(ctx context.Context, req StartJobRequest) error {
	nodeIDs, err := s.nodeDiscoverer.FindNodes(ctx, req.Job)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Debug().Msgf("found %d nodes for job %s", len(nodeIDs), req.Job.Metadata.ID)

	rankedNodes, err := s.nodeRanker.RankNodes(ctx, req.Job, nodeIDs)
	if err != nil {
		return err
	}

	// filter nodes with rank below 0
	var filteredNodes []NodeRank
	for _, node := range rankedNodes {
		if node.Rank >= 0 {
			filteredNodes = append(filteredNodes, node)
		}
	}
	rankedNodes = filteredNodes
	log.Debug().Msgf("ranked %d nodes for job %s", len(rankedNodes), req.Job.Metadata.ID)

	minBids := max(req.Job.Spec.Deal.MinBids, req.Job.Spec.Deal.Concurrency)
	if len(rankedNodes) < minBids {
		return NewErrNotEnoughNodes(minBids, len(rankedNodes))
	}

	sort.Slice(rankedNodes, func(i, j int) bool {
		return rankedNodes[i].Rank > rankedNodes[j].Rank
	})

	err = s.jobStore.AddJob(ctx, &req.Job)
	if err != nil {
		return fmt.Errorf("error saving job id: %w", err)
	}
	s.eventEmitter.EmitJobCreated(ctx, req.Job)

	// TODO: which context should we pass to fsm? We used to pass the request context, but that was wrong
	//  as it the context would be canceled the request returns
	s.shardStateManager.startShardsState(logger.ContextWithNodeIDLogger(context.Background(), s.id), &req.Job, s)
	for _, nodeRank := range rankedNodes[:min(len(rankedNodes), minBids*OverAskForBidsFactor)] {
		// create a new space linked to request context, but call noitfyAskForBid with a new context
		// as the request context will be canceled the request returns
		_, span := s.newSpan(ctx, "askForBid", req.Job.Metadata.ID)
		go s.notifyAskForBid(logger.ContextWithNodeIDLogger(context.Background(), s.id), span, &req.Job, nodeRank.NodeInfo)
	}

	return nil
}

func (s *Scheduler) notifyAskForBid(ctx context.Context, span trace.Span, job *model.Job, nodeInfo model.NodeInfo) {
	defer span.End()
	// TODO: ask to bid on certain shards rather than asking all compute nodes to bid on all shards
	shardIndexes := make([]int, job.Spec.ExecutionPlan.TotalShards)
	for i := 0; i < job.Spec.ExecutionPlan.TotalShards; i++ {
		shardIndexes[i] = i
	}

	request := compute.AskForBidRequest{
		Job:          *job,
		ShardIndexes: shardIndexes,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: nodeInfo.PeerInfo.ID.String(),
		},
	}

	bid, err := s.computeService.AskForBid(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to ask for bid: %+v", request)
	}

	for _, shardResponse := range bid.ShardResponse {
		if shardResponse.Accepted {
			s.eventEmitter.EmitBidReceived(ctx, request, shardResponse)
			shard := model.JobShard{Job: job, Index: shardResponse.ShardIndex}
			shardState, ok := s.shardStateManager.GetShardState(shard)
			if !ok {
				log.Ctx(ctx).Error().Err(err).Msgf("failed to get shard %s state for bid: %s",
					shard, shardResponse.ExecutionID)
				continue
			}
			shardState.bid(ctx, nodeInfo.PeerInfo.ID.String(), shardResponse.ExecutionID)
		}
	}
}

//////////////////////////////
//    Shard fsm handlers    //
//////////////////////////////

func (s *Scheduler) notifyBidAccepted(ctx context.Context, targetNodeID string, executionID string) {
	go func() {
		log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidAccepted for bid: %s", s.id, executionID)
		request := compute.BidAcceptedRequest{
			ExecutionID: executionID,
			RoutingMetadata: compute.RoutingMetadata{
				SourcePeerID: s.id,
				TargetPeerID: targetNodeID,
			},
		}
		response, err := s.computeService.BidAccepted(ctx, request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to notify BidAccepted for bid: %s", executionID)
		} else {
			s.eventEmitter.EmitBidAccepted(ctx, request, response)
		}
	}()
}

func (s *Scheduler) notifyBidRejected(ctx context.Context, targetNodeID string, executionID string) {
	go func() {
		log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidRejected for bid: %s", s.id, executionID)
		request := compute.BidRejectedRequest{
			ExecutionID: executionID,
			RoutingMetadata: compute.RoutingMetadata{
				SourcePeerID: s.id,
				TargetPeerID: targetNodeID,
			},
		}
		response, err := s.computeService.BidRejected(ctx, request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to notify BidRejected for bid: %s", executionID)
		} else {
			s.eventEmitter.EmitBidRejected(ctx, request, response)
		}
	}()
}

func (s *Scheduler) notifyResultAccepted(ctx context.Context, result verifier.VerifierResult) {
	go func() {
		log.Ctx(ctx).Debug().Msgf("Requester node %s responding with ResultAccepted for bid: %s", s.id, result.ExecutionID)
		request := compute.ResultAcceptedRequest{
			ExecutionID: result.ExecutionID,
			RoutingMetadata: compute.RoutingMetadata{
				SourcePeerID: s.id,
				TargetPeerID: result.NodeID,
			},
		}
		response, err := s.computeService.ResultAccepted(ctx, request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to notify ResultAccepted for execution: %s", result.ExecutionID)
		} else {
			s.eventEmitter.EmitResultAccepted(ctx, request, response)
		}
	}()
}

func (s *Scheduler) notifyResultRejected(ctx context.Context, result verifier.VerifierResult) {
	go func() {
		log.Ctx(ctx).Debug().Msgf("Requester node %s responding with ResultRejected for bid: %s", s.id, result.ExecutionID)
		request := compute.ResultRejectedRequest{
			ExecutionID: result.ExecutionID,
			RoutingMetadata: compute.RoutingMetadata{
				SourcePeerID: s.id,
				TargetPeerID: result.NodeID,
			},
		}
		response, err := s.computeService.ResultRejected(ctx, request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to notify ResultRejected for execution: %s", result.ExecutionID)
		} else {
			s.eventEmitter.EmitResultRejected(ctx, request, response)
		}
	}()
}

func (s *Scheduler) notifyCancel(ctx context.Context, message string, nodeID, executionID string) {
	go s.notifyCancelSync(ctx, message, nodeID, executionID)
}

func (s *Scheduler) notifyShardError(ctx context.Context, shard model.JobShard, message string, nodes map[string]string) {
	go func() {
		for nodeID, executionID := range nodes {
			s.notifyCancelSync(ctx, message, nodeID, executionID)
		}
		s.eventEmitter.EmitEventSilently(ctx, model.JobEvent{
			SourceNodeID: s.id,
			JobID:        shard.Job.Metadata.ID,
			ShardIndex:   shard.Index,
			Status:       message,
			EventName:    model.JobEventError,
			EventTime:    time.Now(),
		})
	}()
}

func (s *Scheduler) notifyCancelSync(ctx context.Context, message string, nodeID, executionID string) {
	var err error
	log.Ctx(ctx).Debug().Msgf("Requester node %s canceling%s due to %s", s.id, executionID, message)
	request := compute.CancelExecutionRequest{
		ExecutionID:   executionID,
		Justification: message,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: nodeID,
		},
	}
	_, err = s.computeService.CancelExecution(ctx, request)

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to notify cancellation for execution: %s", executionID)
	}
}

// called for both JobEventShardCompleted and JobEventShardError
// we ask the verifier "IsExecutionComplete" to decide if we can start
// verifying the results - each verifier might have a different
// answer for IsExecutionComplete so we pass off to it to decide
// we mark the job as "verifying" to prevent duplicate verification
func (s *Scheduler) verifyShard(
	ctx context.Context,
	shard model.JobShard,
) ([]verifier.VerifierResult, error) {
	jobVerifier, err := s.verifiers.Get(ctx, shard.Job.Spec.Verifier)
	if err != nil {
		return nil, err
	}
	// ask the verifier if we have enough to start the verification yet
	isExecutionComplete, err := jobVerifier.IsExecutionComplete(ctx, shard)
	if err != nil {
		return nil, err
	}
	if !isExecutionComplete {
		return nil, fmt.Errorf("verifying shard %s but execution is not complete", shard)
	}

	verificationResults, err := jobVerifier.VerifyShard(ctx, shard)
	if err != nil {
		return nil, err
	}

	// we don't fail on first error from the bid queue to avoid a poison pill blocking any progress
	var verifiedResults []verifier.VerifierResult
	// loop over each verification result and publish events
	for _, verificationResult := range verificationResults {
		if verificationResult.Verified {
			s.notifyResultAccepted(ctx, verificationResult)
			verifiedResults = append(verifiedResults, verificationResult)
		} else {
			s.notifyResultRejected(ctx, verificationResult)
		}
	}

	return verifiedResults, nil
}

///////////////////////////////
// Compute callback handlers //
///////////////////////////////

func (s *Scheduler) OnRunComplete(ctx context.Context, result compute.RunResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received RunComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitRunComplete(ctx, result)
	shardState := s.getShardState(ctx, result.ExecutionMetadata)
	if shardState != nil {
		shardState.verifyResult(ctx, result.SourcePeerID, result.ExecutionID)
	}
}

func (s *Scheduler) OnPublishComplete(ctx context.Context, result compute.PublishResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received PublishComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitPublishComplete(ctx, result)
	shardState := s.getShardState(ctx, result.ExecutionMetadata)
	if shardState != nil {
		shardState.resultsPublished(ctx, result.SourcePeerID, result.ExecutionID)
	}
}

func (s *Scheduler) OnCancelComplete(ctx context.Context, result compute.CancelResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received CancelComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	// pass
}

func (s *Scheduler) OnComputeFailure(ctx context.Context, result compute.ComputeError) {
	log.Ctx(ctx).Debug().Err(result).Msgf("Requester node %s received ComputeFailure for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitComputeFailure(ctx, result)
	shardState := s.getShardState(ctx, result.ExecutionMetadata)
	if shardState != nil {
		shardState.computeError(ctx, result.SourcePeerID, result.ExecutionID)
	}
}

func (s *Scheduler) getShardState(ctx context.Context, resultInfo compute.ExecutionMetadata) *shardStateMachine {
	j, err := s.jobStore.GetJob(ctx, resultInfo.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to get job %s when handling callback event for execution %s",
			resultInfo.JobID, resultInfo.ExecutionID)
		return nil
	}
	shard := model.JobShard{Job: j, Index: resultInfo.ShardIndex}
	shardState, ok := s.shardStateManager.GetShardState(shard)
	if !ok {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to get shard %s state for execution: %s",
			shard, resultInfo.ExecutionID)
	}
	return shardState
}

func (s *Scheduler) newSpan(ctx context.Context, name, jobID string) (context.Context, trace.Span) {
	return system.Span(ctx, "requester", name,
		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, s.id),
			attribute.String(model.TracerAttributeNameJobID, jobID),
		),
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// compile-time check that BackendCallback implements the expected interfaces
var _ compute.Callback = (*Scheduler)(nil)
