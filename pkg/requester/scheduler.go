package requester

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/jobstore"
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
	ID               string
	Host             host.Host
	JobStore         jobstore.Store
	NodeDiscoverer   NodeDiscoverer
	NodeRanker       NodeRanker
	ComputeEndpoint  compute.Endpoint
	Verifiers        verifier.VerifierProvider
	StorageProviders storage.StorageProvider
	EventEmitter     EventEmitter
}

type Scheduler struct {
	id               string
	host             host.Host
	jobStore         jobstore.Store
	nodeDiscoverer   NodeDiscoverer
	nodeRanker       NodeRanker
	computeService   compute.Endpoint
	verifiers        verifier.VerifierProvider
	storageProviders storage.StorageProvider
	eventEmitter     EventEmitter
	mu               sync.Mutex
}

func NewScheduler(params SchedulerParams) *Scheduler {
	res := &Scheduler{
		id:               params.ID,
		host:             params.Host,
		jobStore:         params.JobStore,
		nodeDiscoverer:   params.NodeDiscoverer,
		nodeRanker:       params.NodeRanker,
		computeService:   params.ComputeEndpoint,
		verifiers:        params.Verifiers,
		storageProviders: params.StorageProviders,
		eventEmitter:     params.EventEmitter,
	}

	// TODO: replace with job level lock
	res.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "Scheduler.mu",
	})
	return res
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
	log.Ctx(ctx).Debug().Msgf("ranked %d nodes for job %s", len(rankedNodes), req.Job.Metadata.ID)

	minBids := system.Max(req.Job.Spec.Deal.MinBids, req.Job.Spec.Deal.Concurrency)
	if len(rankedNodes) < minBids {
		return NewErrNotEnoughNodes(minBids, len(rankedNodes))
	}

	sort.Slice(rankedNodes, func(i, j int) bool {
		return rankedNodes[i].Rank > rankedNodes[j].Rank
	})

	err = s.jobStore.CreateJob(ctx, req.Job)
	if err != nil {
		return fmt.Errorf("error saving job id: %w", err)
	}
	s.eventEmitter.EmitJobCreated(ctx, req.Job)

	for _, nodeRank := range rankedNodes[:system.Min(len(rankedNodes), minBids*OverAskForBidsFactor)] {
		// create a new space linked to request context, but call noitfyAskForBid with a new context
		// as the request context will be canceled the request returns
		go s.notifyAskForBid(logger.ContextWithNodeIDLogger(context.Background(), s.id), trace.LinkFromContext(ctx), &req.Job, nodeRank.NodeInfo)
	}
	return nil
}

func (s *Scheduler) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received CancelJob for job: %s with reason %s",
		s.id, request.JobID, request.Reason)

	jobState, err := s.jobStore.GetJobState(ctx, request.JobID)
	if err != nil {
		return CancelJobResult{}, err
	}
	var shardIDs []model.ShardID
	for _, shard := range jobState.Shards {
		if !shard.State.IsTerminal() {
			shardIDs = append(shardIDs, shard.ID())
		}
	}
	if len(shardIDs) == 0 {
		return CancelJobResult{}, NewErrJobAlreadyTerminal(request.JobID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, shardID := range shardIDs {
		s.stopShard(ctx, shardID, request.Reason, request.UserTriggered)
	}
	return CancelJobResult{}, nil
}

//////////////////////////////
//    Shard fsm handlers    //
//////////////////////////////

func (s *Scheduler) notifyAskForBid(ctx context.Context, link trace.Link, job *model.Job, nodeInfo model.NodeInfo) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester.Scheduler.StartJob",
		trace.WithLinks(link), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, s.id),
			attribute.String(model.TracerAttributeNameJobID, job.Metadata.ID),
		),
	)
	defer span.End()
	// TODO: ask to bid on certain shards rather than asking all compute nodes to bid on all shards
	shardIndexes := make([]int, job.Spec.ExecutionPlan.TotalShards)
	for i := 0; i < job.Spec.ExecutionPlan.TotalShards; i++ {
		shardIndexes[i] = i
	}

	// persist the intent to ask the node for a bid, which is helpful to avoid asking an unresponsive node again during retries
	for _, shardIndex := range shardIndexes {
		err := s.jobStore.CreateExecution(ctx, model.ExecutionState{
			JobID:      job.Metadata.ID,
			ShardIndex: shardIndex,
			NodeID:     nodeInfo.PeerInfo.ID.String(),
			State:      model.ExecutionStateAskForBid,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error creating execution")
			return
		}
	}

	// do ask for bid
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

	// handle responses
	for _, shardResponse := range bid.ShardResponse {
		s.handleAskForBidResponse(ctx, request, shardResponse)
	}
}

func (s *Scheduler) notifyBidAccepted(ctx context.Context, execution model.ExecutionState) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidAccepted for bid: %s", s.id, execution.ComputeReference)
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: execution.ID(),
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState:   execution.State,
			ExpectedVersion: execution.Version,
		},
		NewValues: model.ExecutionState{
			State: model.ExecutionStateBidAccepted,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to BidAccepted. %s", execution)
	} else {
		go func() {
			request := compute.BidAcceptedRequest{
				ExecutionID: execution.ComputeReference,
				RoutingMetadata: compute.RoutingMetadata{
					SourcePeerID: s.id,
					TargetPeerID: execution.NodeID,
				},
			}
			response, notifyErr := s.computeService.BidAccepted(ctx, request)
			if notifyErr != nil {
				log.Ctx(ctx).Error().Err(notifyErr).Msgf("failed to notify BidAccepted for bid: %s", execution.ComputeReference)
			}
			s.eventEmitter.EmitBidAccepted(ctx, request, response)
		}()
	}
}

func (s *Scheduler) notifyBidRejected(ctx context.Context, execution model.ExecutionState) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidRejected for bid: %s", s.id, execution.ComputeReference)
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: execution.ID(),
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState:   execution.State,
			ExpectedVersion: execution.Version,
		},
		NewValues: model.ExecutionState{
			State: model.ExecutionStateBidRejected,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to BidRejected. %s", execution)
	} else {
		go func() {
			request := compute.BidRejectedRequest{
				ExecutionID: execution.ComputeReference,
				RoutingMetadata: compute.RoutingMetadata{
					SourcePeerID: s.id,
					TargetPeerID: execution.NodeID,
				},
			}
			response, notifyErr := s.computeService.BidRejected(ctx, request)
			if notifyErr != nil {
				log.Ctx(ctx).Error().Err(notifyErr).Msgf("failed to notify BidRejected for bid: %s", execution.ComputeReference)
			}
			s.eventEmitter.EmitBidRejected(ctx, request, response)
		}()
	}
}

func (s *Scheduler) notifyResultAccepted(ctx context.Context, result verifier.VerifierResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with ResultAccepted for bid: %s", s.id, result.Execution.ID())
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: result.Execution.ID(),
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateResultProposed,
		},
		NewValues: model.ExecutionState{
			VerificationResult: model.VerificationResult{
				Complete: true,
				Result:   true,
			},
			State: model.ExecutionStateResultAccepted,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to ResultAccepted. %s", result.Execution.ID())
	} else {
		go func() {
			request := compute.ResultAcceptedRequest{
				ExecutionID: result.Execution.ComputeReference,
				RoutingMetadata: compute.RoutingMetadata{
					SourcePeerID: s.id,
					TargetPeerID: result.Execution.NodeID,
				},
			}
			response, notifyErr := s.computeService.ResultAccepted(ctx, request)
			if notifyErr != nil {
				log.Ctx(ctx).Error().Err(notifyErr).Msgf("failed to notify ResultAccepted for execution: %s", result.Execution.ID())
			}
			s.eventEmitter.EmitResultAccepted(ctx, request, response)
		}()
	}
}

func (s *Scheduler) notifyResultRejected(ctx context.Context, result verifier.VerifierResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with ResultRejected for bid: %s", s.id, result.Execution.ID())
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: result.Execution.ID(),
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateResultProposed,
		},
		NewValues: model.ExecutionState{
			VerificationResult: model.VerificationResult{
				Complete: true,
				Result:   false,
			},
			State: model.ExecutionStateResultRejected,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to ResultRejected. %s", result.Execution.ID())
	} else {
		go func() {
			request := compute.ResultRejectedRequest{
				ExecutionID: result.Execution.ComputeReference,
				RoutingMetadata: compute.RoutingMetadata{
					SourcePeerID: s.id,
					TargetPeerID: result.Execution.NodeID,
				},
			}
			response, notifyErr := s.computeService.ResultRejected(ctx, request)
			if notifyErr != nil {
				log.Ctx(ctx).Error().Err(notifyErr).Msgf("failed to notify ResultRejected for execution: %s", result.Execution.ID())
			}
			s.eventEmitter.EmitResultRejected(ctx, request, response)
		}()
	}
}

// notifyCancel only notifies compute nodes and doesn't update the execution state in the job store. This is because
// the execution state is updated when stopping/failing the shard itself.
func (s *Scheduler) notifyCancel(ctx context.Context, message string, execution model.ExecutionState) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with Cancel for bid: %s", s.id, execution.ComputeReference)
	go func() {
		request := compute.CancelExecutionRequest{
			ExecutionID:   execution.ComputeReference,
			Justification: message,
			RoutingMetadata: compute.RoutingMetadata{
				SourcePeerID: s.id,
				TargetPeerID: execution.NodeID,
			},
		}
		_, err := s.computeService.CancelExecution(ctx, request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to notify cancellation for execution: %s", execution.ComputeReference)
		}
	}()
}

// called for both JobEventShardCompleted and JobEventShardError
// we ask the verifier "IsExecutionComplete" to decide if we can start
// verifying the results - each verifier might have a different
// answer for IsExecutionComplete so we pass off to it to decide
// we mark the job as "verifying" to prevent duplicate verification
func (s *Scheduler) verifyShard(
	ctx context.Context,
	shard model.JobShard,
	executionStates []model.ExecutionState,
) ([]verifier.VerifierResult, error) {
	jobVerifier, err := s.verifiers.Get(ctx, shard.Job.Spec.Verifier)
	if err != nil {
		return nil, err
	}

	verificationResults, err := jobVerifier.VerifyShard(ctx, shard, executionStates)
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

// /////////////////////////////
// Compute callback handlers //
// /////////////////////////////

func (s *Scheduler) handleAskForBidResponse(ctx context.Context,
	request compute.AskForBidRequest,
	shardResponse compute.AskForBidShardResponse) {
	log.Ctx(ctx).Debug().Msgf("Requester node received bid response %+v", shardResponse)

	executionID := model.ExecutionID{
		JobID:      shardResponse.JobID,
		ShardIndex: shardResponse.ShardIndex,
		NodeID:     request.TargetPeerID,
	}
	newState := model.ExecutionStateAskForBidRejected
	if shardResponse.Accepted {
		newState = model.ExecutionStateAskForBidAccepted
	}
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: executionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateAskForBid,
		},
		NewValues: model.ExecutionState{
			ComputeReference: shardResponse.ExecutionID,
			State:            newState,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[handleAskForBidResponse] failed to update execution")
		return
	}

	// decide if we should notify compute node of the bid decision
	// we only notify if we've already received more than MinBids
	if shardResponse.Accepted {
		s.eventEmitter.EmitBidReceived(ctx, request, shardResponse)
		s.startAcceptingBidsIfPossible(ctx, shardResponse)
	} else {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.failIfRecoveryIsNotPossible(ctx, model.ShardID{JobID: shardResponse.JobID, Index: shardResponse.ShardIndex},
			errors.New("not enough bids received"))
	}
}

// startAcceptingBidsIfPossible is called when a compute node has accepted a bid
// If we have received more than MinBids, we start accepting/rejecting bids, and notify the compute node of the decision
func (s *Scheduler) startAcceptingBidsIfPossible(ctx context.Context, shardResponse compute.AskForBidShardResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	shardID := model.ShardID{JobID: shardResponse.JobID, Index: shardResponse.ShardIndex}
	shardState, err := s.jobStore.GetShardState(ctx, shardID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[startAcceptingBidsIfPossible] failed to get shard state")
		return
	}

	var pendingBids []model.ExecutionState
	var receivedBidsCount int
	var activeExecutionsCount int
	for _, execution := range shardState.Executions {
		// total compute nodes that submitted a bid for this shard
		if execution.HasAcceptedAskForBid() {
			receivedBidsCount++
		}
		// compute nodes that are waiting for a decision on their bid
		if execution.State == model.ExecutionStateAskForBidAccepted {
			pendingBids = append(pendingBids, execution)
		}
		// compute nodes that are actively executing the shard, or has completed execution
		if execution.State.IsActive() {
			activeExecutionsCount++
		}
	}

	job, err := s.jobStore.GetJob(ctx, shardResponse.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[startAcceptingBidsIfPossible] failed to get job")
		return
	}
	// if we have more than MinBids, we start selecting the best bids and notify the compute nodes
	if receivedBidsCount >= job.Spec.Deal.MinBids {
		// TODO: we should verify a bid acceptance was received by the compute node before rejecting other bids
		for _, candidate := range pendingBids {
			if activeExecutionsCount < job.Spec.Deal.Concurrency {
				s.notifyBidAccepted(ctx, candidate)
				activeExecutionsCount++
			} else {
				s.notifyBidRejected(ctx, candidate)
			}
		}
	}
}

func (s *Scheduler) OnRunComplete(ctx context.Context, result compute.RunResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received RunComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitRunComplete(ctx, result)

	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: model.ExecutionID{
			JobID:       result.JobID,
			ShardIndex:  result.ShardIndex,
			NodeID:      result.SourcePeerID,
			ExecutionID: result.ExecutionID,
		},
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateBidAccepted,
		},
		NewValues: model.ExecutionState{
			VerificationProposal: result.ResultProposal,
			RunOutput:            result.RunCommandResult,
			State:                model.ExecutionStateResultProposed,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnRunComplete] failed to update execution")
		return
	}

	s.startVerificationIfPossible(ctx, model.ShardID{JobID: result.JobID, Index: result.ShardIndex})
}

func (s *Scheduler) startVerificationIfPossible(ctx context.Context, shardID model.ShardID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// check if we gathered enough results to start verification
	shardState, err := s.jobStore.GetShardState(ctx, shardID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[startVerificationIfPossible] failed to get shard state")
		return
	}

	var pendingVerifications []model.ExecutionState
	for _, execution := range shardState.Executions {
		// compute nodes that are waiting for their results to be verified
		if execution.State == model.ExecutionStateResultProposed {
			pendingVerifications = append(pendingVerifications, execution)
		}
	}

	job, err := s.jobStore.GetJob(ctx, shardID.JobID)
	shard := model.JobShard{Job: &job, Index: shardID.Index}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[startAcceptingBidsIfPossible] failed to get job")
		return
	}
	// TODO: technically we can start verifying if we have enough results compared to deal's confidence
	//  and concurrency. Though we will have ot handle the case where verification fails, but can still
	//  succeed if we wait for more results.
	if len(pendingVerifications) >= job.Spec.Deal.Concurrency {
		verifiedResults, verificationErr := s.verifyShard(ctx, shard, pendingVerifications)
		if verificationErr != nil {
			s.failIfRecoveryIsNotPossible(ctx, shardID, fmt.Errorf("failed to verify shard %s: %w", shardID, verificationErr))
			return
		}
		if len(verifiedResults) == 0 {
			s.failIfRecoveryIsNotPossible(ctx, shardID, fmt.Errorf("failed to verify shard %s: no verified results", shardID))
			return
		}
	}
}

func (s *Scheduler) OnPublishComplete(ctx context.Context, result compute.PublishResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received PublishComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitPublishComplete(ctx, result)
	// TODO: #831 verify that the published results are the same as the ones we expect, or let the verifier
	//  publish the result and not all the compute nodes.

	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: model.ExecutionID{
			JobID:       result.JobID,
			ShardIndex:  result.ShardIndex,
			NodeID:      result.SourcePeerID,
			ExecutionID: result.ExecutionID,
		},
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateResultAccepted,
		},
		NewValues: model.ExecutionState{
			PublishedResult: result.PublishResult,
			State:           model.ExecutionStateCompleted,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnPublishComplete] failed to update execution")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	shardID := model.ShardID{JobID: result.JobID, Index: result.ShardIndex}
	shardState, err := s.jobStore.GetShardState(ctx, shardID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[OnPublishComplete] failed to get shard state")
		return
	}
	if shardState.State == model.ShardStateCompleted {
		// result already published by another node
		return
	}
	err = jobstore.CompleteShard(ctx, s.jobStore, shardID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnPublishComplete] failed to update shard state")
		return
	} else {
		log.Ctx(ctx).Info().Msgf("Shard %s completed successfully", shardID)
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

	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: model.ExecutionID{
			JobID:       result.JobID,
			ShardIndex:  result.ShardIndex,
			NodeID:      result.SourcePeerID,
			ExecutionID: result.ExecutionID,
		},
		Condition: jobstore.UpdateExecutionCondition{
			UnexpectedStates: []model.ExecutionStateType{
				model.ExecutionStateCompleted,
				model.ExecutionStateCanceled,
			},
		},
		NewValues: model.ExecutionState{
			State:  model.ExecutionStateFailed,
			Status: result.Err,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnComputeFailure] failed to update execution")
		return
	}

	s.eventEmitter.EmitComputeFailure(ctx, result)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failIfRecoveryIsNotPossible(ctx, model.ShardID{JobID: result.JobID, Index: result.ShardIndex}, result)
}

// make sure to call this function with the lock held
func (s *Scheduler) failIfRecoveryIsNotPossible(ctx context.Context, shardID model.ShardID, failure error) {
	if !s.isRecoveryStillPossible(ctx, shardID) {
		s.stopShard(ctx, shardID, failure.Error(), false)
	}
}

func (s *Scheduler) isRecoveryStillPossible(ctx context.Context, shardID model.ShardID) bool {
	shardState, err := s.jobStore.GetShardState(ctx, shardID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[isRecoveryStillPossible] failed to get shard state")
		return false
	}
	job, err := s.jobStore.GetJob(ctx, shardID.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[isRecoveryStillPossible] failed to get job")
		return false
	}

	activeExecutions := 0
	for _, execution := range shardState.Executions {
		if !execution.State.IsDiscarded() {
			activeExecutions++
		}
	}

	return activeExecutions >= job.Spec.Deal.Concurrency
}

// make sure to call this function with the lock held
func (s *Scheduler) stopShard(ctx context.Context, shardID model.ShardID, reason string, userRequested bool) {
	if userRequested {
		log.Ctx(ctx).Info().Msgf("stopping shard %s because the user requested it", shardID)
	} else {
		log.Ctx(ctx).Error().Err(errors.New(reason)).Msgf("error completing shard %s", shardID)
	}

	cancelledExecutions, err := jobstore.StopShard(ctx, s.jobStore, shardID, reason, userRequested)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[stopShard] failed to stop shard")
	}

	for _, execution := range cancelledExecutions {
		s.notifyCancel(ctx, reason, execution)
	}
	eventName := model.JobEventError
	if userRequested {
		eventName = model.JobEventCanceled
	}
	s.eventEmitter.EmitEventSilently(ctx, model.JobEvent{
		SourceNodeID: s.id,
		JobID:        shardID.JobID,
		ShardIndex:   shardID.Index,
		Status:       reason,
		EventName:    eventName,
		EventTime:    time.Now(),
	})
}

// compile-time check that BackendCallback implements the expected interfaces
var _ compute.Callback = (*Scheduler)(nil)
