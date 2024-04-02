package planner

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

type ComputeForwarder struct {
	id             string
	computeService compute.Endpoint
	jobStore       jobstore.Store
	evalBroker     orchestrator.EvaluationBroker //nolint:unused
}

type ComputeForwarderParams struct {
	ID             string
	ComputeService compute.Endpoint
	JobStore       jobstore.Store
}

func NewComputeForwarder(params ComputeForwarderParams) *ComputeForwarder {
	return &ComputeForwarder{
		id:             params.ID,
		computeService: params.ComputeService,
		jobStore:       params.JobStore,
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
		waitForApproval := exec.DesiredState.StateType == models.ExecutionDesiredStatePending
		s.doNotifyAskForBid(ctx, exec, waitForApproval)
	}
	for _, u := range plan.UpdatedExecutions {
		observedState := u.Execution.ComputeState.StateType

		switch u.DesiredState {
		case models.ExecutionDesiredStatePending:
			if observedState == models.ExecutionStateNew {
				s.doNotifyAskForBid(ctx, u.Execution, true)
			}
		case models.ExecutionDesiredStateRunning:
			if observedState == models.ExecutionStateAskForBidAccepted {
				s.doNotifyBidAccepted(ctx, u.Execution)
			}
			if observedState == models.ExecutionStateNew {
				s.doNotifyAskForBid(ctx, u.Execution, false)
			}
		case models.ExecutionDesiredStateStopped:
			if observedState == models.ExecutionStateAskForBidAccepted {
				s.doNotifyBidRejected(ctx, u.Execution)
			} else if !u.Execution.IsTerminalComputeState() {
				s.notifyCancel(ctx, u.Comment, u.Execution)
			}
		}
	}
}

// doNotifyAskForBid notifies the target node to bid for the given execution.
func (s *ComputeForwarder) doNotifyAskForBid(ctx context.Context, execution *models.Execution, waitForApproval bool) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s asking node %s to bid with execution %s",
		s.id, execution.NodeID, execution.ID)

	// Notify the compute node
	request := compute.AskForBidRequest{
		Execution:       execution,
		WaitForApproval: waitForApproval,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: execution.NodeID,
		},
	}
	_, err := s.computeService.AskForBid(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s to bid for execution %s",
			execution.NodeID, execution.ID)
		return
	}

	// Update the execution state if the notification was successful
	newState := models.ExecutionStateAskForBid
	if !waitForApproval {
		newState = models.ExecutionStateBidAccepted
	}
	s.updateExecutionState(ctx, execution, newState, models.ExecutionStateNew)
}

// doNotifyBidAccepted notifies the target node that the bid was accepted.
func (s *ComputeForwarder) doNotifyBidAccepted(ctx context.Context, execution *models.Execution) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidAccepted for bid: %s", s.id, execution.ID)
	request := compute.BidAcceptedRequest{
		ExecutionID: execution.ID,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: execution.NodeID,
		},
	}
	_, err := s.computeService.BidAccepted(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s that bid %s was accepted",
			execution.NodeID, execution.ID)
		return
	}

	// Update the execution state if the notification was successful
	s.updateExecutionState(ctx, execution, models.ExecutionStateBidAccepted, models.ExecutionStateAskForBidAccepted)
}

// doNotifyBidRejected notifies the target node that the bid was rejected.
func (s *ComputeForwarder) doNotifyBidRejected(ctx context.Context, execution *models.Execution) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with BidRejected for bid: %s", s.id, execution.ID)
	request := compute.BidRejectedRequest{
		ExecutionID: execution.ID,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: execution.NodeID,
		},
	}
	_, err := s.computeService.BidRejected(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s that bid %s was rejected",
			execution.NodeID, execution.ID)
		return
	}

	// Update the execution state if the notification was successful
	s.updateExecutionState(ctx, execution, models.ExecutionStateBidRejected, models.ExecutionStateAskForBidAccepted)
}

// notifyCancel notifies the target node that the execution was canceled.
func (s *ComputeForwarder) notifyCancel(ctx context.Context, message string, execution *models.Execution) {
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with Cancel for bid: %s", s.id, execution.ID)

	request := compute.CancelExecutionRequest{
		ExecutionID:   execution.ID,
		Justification: message,
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: s.id,
			TargetPeerID: execution.NodeID,
		},
	}
	_, err := s.computeService.CancelExecution(ctx, request)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s that execution %s was canceled",
			execution.NodeID, execution.ID)
		return
	}

	// Update the execution state even if the notification failed
	s.updateExecutionState(ctx, execution, models.ExecutionStateCancelled)
}

// updateExecutionState updates the execution state in the job store.
func (s *ComputeForwarder) updateExecutionState(ctx context.Context, execution *models.Execution,
	newState models.ExecutionStateType, expectedStates ...models.ExecutionStateType) {
	err := s.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues: models.Execution{
			ComputeState: models.State[models.ExecutionStateType]{
				StateType: newState,
			},
		},
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedStates: expectedStates,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to update execution %s to state %s", execution.ID, newState)
	}
}
