package requester

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
)

// transitionJobState checks the current state of the job and transitions it to the next state if possible, along with triggering
// actions needed to transition, such as updating the job state and notifying compute nodes.
// This method is agnostic to how it was called to allow using the same logic as a response to callback from a compute node, or
// as a result of a periodic check that checks for stale jobs.
func (s *BaseScheduler) transitionJobState(ctx context.Context, jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transitionJobStateLockFree(ctx, jobID)
}

func (s *BaseScheduler) transitionJobStateLockFree(ctx context.Context, jobID string) {
	jobState, err := s.jobStore.GetJobState(ctx, jobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[transitionJobState] failed to get job state")
		return
	}

	if jobState.State.IsTerminal() {
		log.Ctx(ctx).Debug().Msgf("[transitionJobState] job %s is already in terminal state %s", jobID, jobState.State)
		return
	}

	job, err := s.jobStore.GetJob(ctx, jobID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("[transitionJobState] failed to get job")
		return
	}

	s.checkForFailedExecutions(ctx, job, jobState)
	s.checkForPendingBids(ctx, job, jobState)
	s.checkForPendingResults(ctx, job, jobState)
	s.checkForCompletedExecutions(ctx, job, jobState)
}

// checkForFailedExecutions checks if any execution has failed and if so, check if executions can be retried,
// or transitions the job to a failed state.
func (s *BaseScheduler) checkForFailedExecutions(ctx context.Context, job model.Job, jobState model.JobState) {
	var receivedBidsCount int
	var publishedOrPublishingCount int
	var nonDiscardedExecutionsCount int
	var lastFailedExecution model.ExecutionState
	for _, execution := range jobState.Executions {
		if execution.HasAcceptedAskForBid() {
			receivedBidsCount++
		}
		if execution.State == model.ExecutionStateCompleted || execution.State == model.ExecutionStateResultAccepted {
			publishedOrPublishingCount++
		}
		if !execution.State.IsDiscarded() {
			nonDiscardedExecutionsCount++
		}
		if execution.State == model.ExecutionStateFailed && lastFailedExecution.UpdateTime.Before(execution.UpdateTime) {
			lastFailedExecution = execution
		}
	}

	// calculate how many executions we still need, and evaluate if we can ask more nodes to bid
	var minExecutions int
	if job.Spec.Deal.MinBids > 0 && receivedBidsCount < job.Spec.Deal.MinBids {
		// if we are still queuing bids, then we need at least MinBids to start accepting bids
		minExecutions = system.Max(job.Spec.Deal.GetConcurrency(), job.Spec.Deal.MinBids)
	} else if publishedOrPublishingCount > 0 {
		// if at least a single execution was published or still publishing, then we don't need to retry in case some executions failed to publish
		minExecutions = 1
	} else {
		// by default, we need executions as many as the job concurrency
		minExecutions = job.Spec.Deal.GetConcurrency()
	}

	if nonDiscardedExecutionsCount < minExecutions {
		var finalErr error = nil
		retried := false
		defer func() {
			if !retried {
				if lastFailedExecution.NodeID != "" {
					finalErr = multierr.Append(
						finalErr,
						fmt.Errorf("node %s failed due to: %s", lastFailedExecution.NodeID, lastFailedExecution.Status),
					)
				}

				errMsg := ""
				if finalErr != nil {
					errMsg = finalErr.Error()
				}
				s.stopJob(ctx, job.ID(), errMsg, false)
			}
		}()
		if s.retryStrategy.ShouldRetry(ctx, RetryRequest{JobID: job.ID()}) {
			desiredNodeCount := minExecutions - nonDiscardedExecutionsCount
			rankedNodes, err := s.nodeSelector.SelectNodes(ctx, job, desiredNodeCount, desiredNodeCount)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("[transitionJobState] failed to find enough nodes to retry")
				finalErr = err // So the deferred function can use it for the jobstate
				return
			}
			s.notifyAskForBid(ctx, trace.LinkFromContext(ctx), job, rankedNodes[:desiredNodeCount])
			retried = true
			return
		}
	}
}

// checkForPendingBids checks if any bid is still pending a response, if minBids criteria is met, and accept/reject bids accordingly.
func (s *BaseScheduler) checkForPendingBids(ctx context.Context, job model.Job, jobState model.JobState) {
	executionsByState := jobState.GroupExecutionsByState()
	var receivedBidsCount int
	var activeExecutionsCount int
	for _, execution := range jobState.Executions {
		if execution.HasAcceptedAskForBid() {
			receivedBidsCount++
		}
		if execution.State.IsActive() {
			activeExecutionsCount++
		}
	}

	if receivedBidsCount >= job.Spec.Deal.MinBids {
		// TODO: we should verify a bid acceptance was received by the compute node before rejecting other bids
		for _, candidate := range executionsByState[model.ExecutionStateAskForBidAccepted] {
			if activeExecutionsCount < job.Spec.Deal.Concurrency {
				s.updateAndNotifyBidAccepted(ctx, candidate)
				activeExecutionsCount++
			} else {
				s.updateAndNotifyBidRejected(ctx, candidate)
			}
		}
	}
}

// checkForPendingResults checks if enough executions proposed a result, verify the results, and accept/reject results accordingly.
func (s *BaseScheduler) checkForPendingResults(ctx context.Context, job model.Job, jobState model.JobState) {
	executionsByState := jobState.GroupExecutionsByState()
	if len(executionsByState[model.ExecutionStateResultProposed]) >= job.Spec.Deal.Concurrency {
		verifiedResults, verificationErr := s.verifyResult(ctx, job, executionsByState[model.ExecutionStateResultProposed])
		if verificationErr != nil {
			s.stopJob(ctx, job.ID(), fmt.Sprintf("failed to verify job %s: %s", job.ID(), verificationErr), false)
			return
		}
		// re-trigger job state transition if verification failed to see if we can retry and recover
		if len(verifiedResults) == 0 {
			s.transitionJobStateLockFree(ctx, job.ID())
		}
	}
}

// checkForPendingPublishing checks if all verified executions have published, and if so, transition the job to a completed state.
func (s *BaseScheduler) checkForCompletedExecutions(ctx context.Context, job model.Job, jobState model.JobState) {
	var completedCount int
	var stillPublishing bool
	for _, execution := range jobState.Executions {
		if execution.State == model.ExecutionStateCompleted {
			completedCount++
		}
		if execution.State == model.ExecutionStateResultAccepted {
			stillPublishing = true
		}
	}
	// no action to take if we have not published all verified results yet
	if stillPublishing {
		return
	}
	if completedCount > 0 {
		newState := model.JobStateCompleted
		if completedCount < job.Spec.Deal.GetConfidence() {
			newState = model.JobStateCompletedPartially
		}
		err := s.jobStore.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
			JobID:    job.ID(),
			NewState: newState,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("[checkForCompletedExecutions] failed to update job state")
			return
		} else {
			msg := fmt.Sprintf("job %s completed successfully", job.ID())
			if newState == model.JobStateCompletedPartially {
				msg += " partially with some failed executions"
			}
			log.Ctx(ctx).Info().Msg(msg)
		}
	}
}
