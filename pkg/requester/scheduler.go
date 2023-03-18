package requester

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
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

type scheduler struct {
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

func NewScheduler(params SchedulerParams) *scheduler {
	res := &scheduler{
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

func (s *scheduler) StartJob(ctx context.Context, req StartJobRequest) error {
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

	err = s.jobStore.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
		JobID: req.Job.Metadata.ID,
		Condition: jobstore.UpdateJobCondition{
			ExpectedState: model.JobStateNew,
		},
		NewState: model.JobStateInProgress,
	})
	if err != nil {
		return errors.Wrap(err, "error saving job id")
	}
	s.eventEmitter.EmitJobCreated(ctx, req.Job)

	selectedNodes := rankedNodes[:system.Min(len(rankedNodes), minBids*OverAskForBidsFactor)]
	go s.notifyAskForBid(logger.ContextWithNodeIDLogger(context.Background(), s.id), trace.LinkFromContext(ctx), &req.Job, selectedNodes)
	return nil
}

func (s *scheduler) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received CancelJob for job: %s with reason %s",
		s.id, request.JobID, request.Reason)

	jobState, err := s.jobStore.GetJobState(ctx, request.JobID)
	if err != nil {
		return CancelJobResult{}, err
	}
	if jobState.State.IsTerminal() {
		return CancelJobResult{}, NewErrJobAlreadyTerminal(request.JobID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopJob(ctx, jobState.JobID, request.Reason, request.UserTriggered)
	return CancelJobResult{}, nil
}

//////////////////////////////
//    Job fsm handlers    //
//////////////////////////////

func (s *scheduler) notifyAskForBid(ctx context.Context, link trace.Link, job *model.Job, nodes []NodeRank) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester.Scheduler.StartJob",
		trace.WithLinks(link), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, s.id),
			attribute.String(model.TracerAttributeNameJobID, job.Metadata.ID),
		),
	)
	defer span.End()

	// persist the intent to ask the node for a bid, which is helpful to avoid asking an unresponsive node again during retries.
	// we persist the intent for all nodes before asking any node to bid, so that we don't fail the job if the first node we ask rejects the
	// the bid before we persist the intent to ask the other nodes.
	for _, node := range nodes {
		err := s.jobStore.CreateExecution(ctx, model.ExecutionState{
			JobID:  job.Metadata.ID,
			NodeID: node.NodeInfo.PeerInfo.ID.String(),
			State:  model.ExecutionStateAskForBid,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error creating execution")
			return
		}
	}

	newCtx := util.NewDetachedContext(ctx)
	for _, node := range nodes {
		go s.doNotifyAskForBid(newCtx, link, job, node.NodeInfo)
	}
}

func (s *scheduler) doNotifyAskForBid(ctx context.Context, link trace.Link, job *model.Job, nodeInfo model.NodeInfo) {
	request := compute.AskForBidRequest{
		Job: *job,
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
	s.handleAskForBidResponse(ctx, request, bid)
}

func (s *scheduler) notifyBidAccepted(ctx context.Context, execution model.ExecutionState) {
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
		newCtx := util.NewDetachedContext(ctx)
		go func(ctx context.Context) {
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
		}(newCtx)
	}
}

func (s *scheduler) notifyBidRejected(ctx context.Context, execution model.ExecutionState) {
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
		newCtx := util.NewDetachedContext(ctx)
		go func(ctx context.Context) {
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
		}(newCtx)
	}
}

func (s *scheduler) notifyResultAccepted(ctx context.Context, result verifier.VerifierResult) {
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
		newCtx := util.NewDetachedContext(ctx)
		go func(ctx context.Context) {
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
		}(newCtx)
	}
}

func (s *scheduler) notifyResultRejected(ctx context.Context, result verifier.VerifierResult) {
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
		newCtx := util.NewDetachedContext(ctx)
		go func(ctx context.Context) {
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
		}(newCtx)
	}
}

// notifyCancel only notifies compute nodes and doesn't update the execution state in the job store. This is because
// the execution state is updated when stopping/failing the job itself.
func (s *scheduler) notifyCancel(ctx context.Context, message string, execution model.ExecutionState) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with Cancel for bid: %s", s.id, execution.ComputeReference)

	newCtx := util.NewDetachedContext(ctx)
	go func(ctx context.Context) {
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
	}(newCtx)
}

// we ask the verifier "IsExecutionComplete" to decide if we can start
// verifying the results - each verifier might have a different
// answer for IsExecutionComplete so we pass off to it to decide
// we mark the job as "verifying" to prevent duplicate verification
func (s *scheduler) verifyResult(
	ctx context.Context,
	job model.Job,
	executionStates []model.ExecutionState,
) ([]verifier.VerifierResult, error) {
	jobVerifier, err := s.verifiers.Get(ctx, job.Spec.Verifier)
	if err != nil {
		return nil, err
	}

	verificationResults, err := jobVerifier.Verify(ctx, job, executionStates)
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

func (s *scheduler) handleAskForBidResponse(ctx context.Context,
	request compute.AskForBidRequest,
	response compute.AskForBidResponse) {
	log.Ctx(ctx).Debug().Msgf("Requester node received bid response %+v", response)

	executionID := model.ExecutionID{
		JobID:  response.JobID,
		NodeID: request.TargetPeerID,
	}
	newState := model.ExecutionStateAskForBidRejected
	if response.Accepted {
		newState = model.ExecutionStateAskForBidAccepted
	}
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: executionID,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: model.ExecutionStateAskForBid,
		},
		NewValues: model.ExecutionState{
			ComputeReference: response.ExecutionID,
			State:            newState,
			Status:           response.Reason,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[handleAskForBidResponse] failed to update execution")
		return
	}

	// decide if we should notify compute node of the bid decision
	// we only notify if we've already received more than MinBids
	if response.Accepted {
		s.eventEmitter.EmitBidReceived(ctx, request, response)
		s.startAcceptingBidsIfPossible(ctx, response)
	} else {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.failIfRecoveryIsNotPossible(ctx, response.JobID, errors.New("not enough bids received"))
	}
}

// startAcceptingBidsIfPossible is called when a compute node has accepted a bid
// If we have received more than MinBids, we start accepting/rejecting bids, and notify the compute node of the decision
func (s *scheduler) startAcceptingBidsIfPossible(ctx context.Context, response compute.AskForBidResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobState, err := s.jobStore.GetJobState(ctx, response.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[startAcceptingBidsIfPossible] failed to get job state")
		return
	}

	var pendingBids []model.ExecutionState
	var receivedBidsCount int
	var activeExecutionsCount int
	for _, execution := range jobState.Executions {
		// total compute nodes that submitted a bid for this job
		if execution.HasAcceptedAskForBid() {
			receivedBidsCount++
		}
		// compute nodes that are waiting for a decision on their bid
		if execution.State == model.ExecutionStateAskForBidAccepted {
			pendingBids = append(pendingBids, execution)
		}
		// compute nodes that are actively executing the job, or has completed execution
		if execution.State.IsActive() {
			activeExecutionsCount++
		}
	}

	job, err := s.jobStore.GetJob(ctx, response.JobID)
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

func (s *scheduler) OnRunComplete(ctx context.Context, result compute.RunResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received RunComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitRunComplete(ctx, result)

	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: model.ExecutionID{
			JobID:       result.JobID,
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

	s.startVerificationIfPossible(ctx, result.JobID)
}

func (s *scheduler) startVerificationIfPossible(ctx context.Context, jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// check if we gathered enough results to start verification
	jobState, err := s.jobStore.GetJobState(ctx, jobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[startVerificationIfPossible] failed to get job state")
		return
	}

	var pendingVerifications []model.ExecutionState
	for _, execution := range jobState.Executions {
		// compute nodes that are waiting for their results to be verified
		if execution.State == model.ExecutionStateResultProposed {
			pendingVerifications = append(pendingVerifications, execution)
		}
	}

	job, err := s.jobStore.GetJob(ctx, jobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[startAcceptingBidsIfPossible] failed to get job")
		return
	}
	// TODO: technically we can start verifying if we have enough results compared to deal's confidence
	//  and concurrency. Though we will have ot handle the case where verification fails, but can still
	//  succeed if we wait for more results.
	if len(pendingVerifications) >= job.Spec.Deal.Concurrency {
		verifiedResults, verificationErr := s.verifyResult(ctx, job, pendingVerifications)
		if verificationErr != nil {
			s.failIfRecoveryIsNotPossible(ctx, jobID, fmt.Errorf("failed to verify job %s: %w", jobID, verificationErr))
			return
		}
		if len(verifiedResults) == 0 {
			s.failIfRecoveryIsNotPossible(ctx, jobID, fmt.Errorf("failed to verify job %s: no verified results", jobID))
			return
		}
	}
}

func (s *scheduler) OnPublishComplete(ctx context.Context, result compute.PublishResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received PublishComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	s.eventEmitter.EmitPublishComplete(ctx, result)
	// TODO: #831 verify that the published results are the same as the ones we expect, or let the verifier
	//  publish the result and not all the compute nodes.

	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: model.ExecutionID{
			JobID:       result.JobID,
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
	jobState, err := s.jobStore.GetJobState(ctx, result.JobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[OnPublishComplete] failed to get job state")
		return
	}
	if !jobState.ExecutionsInTerminalState() {
		// An execution is still being worked on, so the job isn't completed yet.
		return
	}
	err = s.jobStore.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
		JobID:    result.JobID,
		NewState: model.JobStateCompleted,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[OnPublishComplete] failed to update job state")
		return
	} else {
		log.Ctx(ctx).Info().Msgf("Job %s completed successfully", result.JobID)
	}
}

func (s *scheduler) OnCancelComplete(ctx context.Context, result compute.CancelResult) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s received CancelComplete for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)
	// pass
}

func (s *scheduler) OnComputeFailure(ctx context.Context, result compute.ComputeError) {
	log.Ctx(ctx).Debug().Err(result).Msgf("Requester node %s received ComputeFailure for execution: %s from %s",
		s.id, result.ExecutionID, result.SourcePeerID)

	// update execution state
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: model.ExecutionID{
			JobID:       result.JobID,
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
	s.failIfRecoveryIsNotPossible(ctx, result.JobID, result)
}

// make sure to call this function with the lock held
func (s *scheduler) failIfRecoveryIsNotPossible(ctx context.Context, jobID string, failure error) {
	if !s.isRecoveryStillPossible(ctx, jobID) {
		s.stopJob(ctx, jobID, failure.Error(), false)
	}
}

func (s *scheduler) isRecoveryStillPossible(ctx context.Context, jobID string) bool {
	jobState, err := s.jobStore.GetJobState(ctx, jobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[isRecoveryStillPossible] failed to get job state")
		return false
	}
	job, err := s.jobStore.GetJob(ctx, jobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[isRecoveryStillPossible] failed to get job")
		return false
	}

	activeExecutions := 0
	for _, execution := range jobState.Executions {
		if !execution.State.IsDiscarded() {
			activeExecutions++
		}
	}

	return activeExecutions >= job.Spec.Deal.Concurrency
}

// make sure to call this function with the lock held
func (s *scheduler) stopJob(ctx context.Context, jobID, reason string, userRequested bool) {
	if userRequested {
		log.Ctx(ctx).Info().Msgf("stopping job %s because the user requested it", jobID)
	} else {
		log.Ctx(ctx).Error().Err(errors.New(reason)).Msgf("error completing job %s", jobID)
	}

	cancelledExecutions, err := jobstore.StopJob(ctx, s.jobStore, jobID, reason, userRequested)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("[stopJob] failed to stop job")
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
		JobID:        jobID,
		Status:       reason,
		EventName:    eventName,
		EventTime:    time.Now(),
	})
}

// compile-time check that BackendCallback implements the expected interfaces
var _ Scheduler = (*scheduler)(nil)
var _ compute.Callback = (*scheduler)(nil)
