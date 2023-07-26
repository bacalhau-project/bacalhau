package planner

import (
	"context"
	"crypto/rsa"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/secrets"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/rs/zerolog/log"
)

type ComputeForwarder struct {
	id             string
	computeService compute.Endpoint
	jobStore       jobstore.Store
	evalBroker     orchestrator.EvaluationBroker //nolint:unused
	peerStore      peerstore.Peerstore
}

type ComputeForwarderParams struct {
	ID             string
	ComputeService compute.Endpoint
	JobStore       jobstore.Store
	PeerStore      peerstore.Peerstore
}

func NewComputeForwarder(params ComputeForwarderParams) *ComputeForwarder {
	return &ComputeForwarder{
		id:             params.ID,
		computeService: params.ComputeService,
		jobStore:       params.JobStore,
		peerStore:      params.PeerStore,
	}
}

func (s *ComputeForwarder) Process(ctx context.Context, plan *models.Plan) error {
	// notifying nodes is happening in the background, so we can return immediately and not block
	// the scheduler.
	// TODO: we need to handle failed notifications. Either enqueue fail the execution and enqueue an evaluation, or
	//  have a background job that monitors stuck executions and enqueues an evaluation.
	// TODO: would be nice to notify compute nodes async without blocking the scheduler, but we need to
	//  serialize the notifications for the same job/execution, which is now done by the evaluation broker.
	s.doProcess(ctx, plan)
	return nil
}

// doProcess performs the actual notifications to compute nodes based on the plan's desired state.
func (s *ComputeForwarder) doProcess(ctx context.Context, plan *models.Plan) {
	// TODO: notifying nodes for the same plan can be done in parallel, but we should limit
	//  the total number of concurrent notifications across plans to avoid overloading the network.
	for _, exec := range plan.NewExecutions {
		s.doNotifyAskForBid(ctx, *plan.Job, exec.ID())
	}
	for _, u := range plan.UpdatedExecutions {
		observedState := u.Execution.State

		switch u.DesiredState {
		case model.ExecutionDesiredStatePending:
			if observedState == model.ExecutionStateNew {
				s.doNotifyAskForBid(ctx, *plan.Job, u.Execution.ID())
			}
		case model.ExecutionDesiredStateRunning:
			if observedState == model.ExecutionStateAskForBidAccepted {
				s.doNotifyBidAccepted(ctx, u.Execution.ID())
			}
		case model.ExecutionDesiredStateStopped:
			if observedState == model.ExecutionStateAskForBidAccepted {
				s.doNotifyBidRejected(ctx, u.Execution.ID())
			} else if !observedState.IsTerminal() {
				s.notifyCancel(ctx, u.Comment, u.Execution.ID())
			}
		}
	}
}

// doNotifyAskForBid notifies the target node to bid for the given execution.
func (s *ComputeForwarder) doNotifyAskForBid(ctx context.Context, job model.Job, executionID model.ExecutionID) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s asking node %s to bid with executionID %s",
		s.id, executionID.NodeID, executionID.ExecutionID)

	// Check and recrypt any necessary variables
	if secrets.JobNeedsDecrypting(job.Spec) {
		var requesterPrivateKey crypto.PrivKey
		var computePublicKey crypto.PubKey

		for _, id := range s.peerStore.PeersWithKeys() {
			if id.String() == executionID.NodeID {
				computePublicKey = s.peerStore.PubKey(id)
			} else if id.String() == s.id {
				requesterPrivateKey = s.peerStore.PrivKey(id)
			}
		}

		reqKey, _ := crypto.PrivKeyToStdKey(requesterPrivateKey)
		reqPrivKey, _ := reqKey.(*rsa.PrivateKey)

		compKey, _ := crypto.PubKeyToStdKey(computePublicKey)
		computePubKey, _ := compKey.(*rsa.PublicKey)

		job.Spec, _ = secrets.RecryptEnv(job.Spec, reqPrivKey, s.id, computePubKey, executionID.NodeID)
	}

	// Notify the compute node
	request := compute.AskForBidRequest{
		Job: job,
		ExecutionMetadata: compute.ExecutionMetadata{
			ExecutionID: executionID.ExecutionID,
			JobID:       executionID.JobID,
		},
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: executionID.NodeID,
		},
	}
	_, err := s.computeService.AskForBid(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s to bid for execution %s",
			executionID.NodeID, executionID.ExecutionID)
		return
	}

	// Update the execution state if the notification was successful
	s.updateExecutionState(ctx, executionID, model.ExecutionStateAskForBid, model.ExecutionStateNew)
}

// doNotifyBidAccepted notifies the target node that the bid was accepted.
func (s *ComputeForwarder) doNotifyBidAccepted(ctx context.Context, executionID model.ExecutionID) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidAccepted for bid: %s", s.id, executionID.ExecutionID)
	request := compute.BidAcceptedRequest{
		ExecutionID: executionID.ExecutionID,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: executionID.NodeID,
		},
	}
	_, err := s.computeService.BidAccepted(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s that bid %s was accepted",
			executionID.NodeID, executionID.ExecutionID)
		return
	}

	// Update the execution state if the notification was successful
	s.updateExecutionState(ctx, executionID, model.ExecutionStateBidAccepted, model.ExecutionStateAskForBidAccepted)
}

// doNotifyBidRejected notifies the target node that the bid was rejected.
func (s *ComputeForwarder) doNotifyBidRejected(ctx context.Context, executionID model.ExecutionID) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidRejected for bid: %s", s.id, executionID.ExecutionID)
	request := compute.BidRejectedRequest{
		ExecutionID: executionID.ExecutionID,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: executionID.NodeID,
		},
	}
	_, err := s.computeService.BidRejected(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s that bid %s was rejected",
			executionID.NodeID, executionID.ExecutionID)
		return
	}

	// Update the execution state if the notification was successful
	s.updateExecutionState(ctx, executionID, model.ExecutionStateBidRejected, model.ExecutionStateAskForBidAccepted)
}

// notifyCancel notifies the target node that the execution was cancelled.
func (s *ComputeForwarder) notifyCancel(ctx context.Context, message string, executionID model.ExecutionID) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with Cancel for bid: %s", s.id, executionID.ExecutionID)

	request := compute.CancelExecutionRequest{
		ExecutionID:   executionID.ExecutionID,
		Justification: message,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: executionID.NodeID,
		},
	}
	_, err := s.computeService.CancelExecution(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s that execution %s was cancelled",
			executionID.NodeID, executionID.ExecutionID)
		return
	}

	// Update the execution state even if the notification failed
	s.updateExecutionState(ctx, executionID, model.ExecutionStateCancelled)
}

// updateExecutionState updates the execution state in the job store.
func (s *ComputeForwarder) updateExecutionState(ctx context.Context, executionID model.ExecutionID,
	newState model.ExecutionStateType, expectedStates ...model.ExecutionStateType) {
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: executionID,
		NewValues: model.ExecutionState{
			State: newState,
		},
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedStates: expectedStates,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to update execution %s to state %s", executionID.ExecutionID, newState)
	}
}
