package requester

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
)

// TransitionJobState checks the current state of the job and transitions it to the next state if possible, along with triggering
// actions needed to transition, such as updating the job state and notifying compute nodes.
// This method is agnostic to how it was called to allow using the same logic as a response to callback from a compute node, or
// as a result of a periodic check that checks for stale jobs.
func (s *BaseScheduler) TransitionJobState(ctx context.Context, jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transitionJobStateLockFree(ctx, jobID)
}

func (s *BaseScheduler) transitionJobStateLockFree(ctx context.Context, jobID string) {
	ctx = log.Ctx(ctx).With().Str("JobID", jobID).Logger().WithContext(ctx)

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
	nodesToRetry, err := s.nodeSelector.SelectNodesForRetry(ctx, &job, &jobState)
	if err != nil || (len(nodesToRetry) > 0 && !s.retryStrategy.ShouldRetry(ctx, RetryRequest{JobID: job.ID()})) {
		// There was an error selecting nodes, or we need to retry some but retry strategy says no.
		log.Ctx(ctx).Debug().Err(err).Int("NodesToRetry", len(nodesToRetry)).Msg("Failing job")

		var finalErr error
		var errMsg string

		var lastFailedExecution model.ExecutionState
		for _, execution := range jobState.Executions {
			if execution.State == model.ExecutionStateFailed && lastFailedExecution.UpdateTime.Before(execution.UpdateTime) {
				lastFailedExecution = execution
				finalErr = multierr.Append(
					finalErr,
					fmt.Errorf("node %s failed due to: %s", lastFailedExecution.NodeID, lastFailedExecution.Status),
				)
				errMsg = finalErr.Error()
			}
		}

		s.stopJob(ctx, job.ID(), errMsg, false)
	} else if len(nodesToRetry) > 0 {
		// There was no error and retry strategy said yes.
		s.notifyAskForBid(ctx, trace.LinkFromContext(ctx), job, nodesToRetry)
	}
}

// checkForPendingBids checks if any bid is still pending a response
func (s *BaseScheduler) checkForPendingBids(ctx context.Context, job model.Job, jobState model.JobState) {
	acceptBids, rejectBids := s.nodeSelector.SelectBids(ctx, &job, &jobState)
	for _, bid := range acceptBids {
		s.updateAndNotifyBidAccepted(ctx, bid)
	}
	for _, bid := range rejectBids {
		s.updateAndNotifyBidRejected(ctx, bid)
	}
}

// checkForPendingResults checks if enough executions proposed a result, verify the results, and accept/reject results accordingly.
func (s *BaseScheduler) checkForPendingResults(ctx context.Context, job model.Job, jobState model.JobState) {
	if s.nodeSelector.CanVerifyJob(ctx, &job, &jobState) {
		executionsByState := jobState.GroupExecutionsByState()
		succeeded, failed, err := s.verifyResult(ctx, job, executionsByState[model.ExecutionStateResultProposed])
		log.Ctx(ctx).Debug().Err(err).Int("Succeeded", len(succeeded)).Int("Failed", len(failed)).Msg("Attempted to verify results")
		if err != nil {
			s.stopJob(ctx, job.ID(), fmt.Sprintf("failed to verify job %s: %s", job.ID(), err), false)
			return
		}
		if len(failed) > 0 {
			s.transitionJobStateLockFree(ctx, job.ID())
		}
	}
}

// checkForPendingPublishing checks if all verified executions have published, and if so, transition the job to a completed state.
func (s *BaseScheduler) checkForCompletedExecutions(ctx context.Context, job model.Job, jobState model.JobState) {
	if len(jobState.GroupExecutionsByState()[model.ExecutionStateResultAccepted]) > 0 {
		// Some executions are still publishing – come back later.
		return
	}

	shouldUpdate, newState := s.nodeSelector.CanCompleteJob(ctx, &job, &jobState)
	if shouldUpdate {
		err := s.jobStore.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
			JobID:    job.ID(),
			NewState: newState,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("[checkForCompletedExecutions] failed to update job state")
			return
		} else {
			s.eventEmitter.EmitEventSilently(ctx, model.JobEvent{
				SourceNodeID: s.id,
				JobID:        job.ID(),
				EventName:    model.JobEventCompleted,
				EventTime:    time.Now(),
			})

			msg := fmt.Sprintf("job %s completed", job.ID())
			if newState == model.JobStateCompletedPartially {
				msg += " partially; some executions failed to publish results"
			} else {
				msg += " successfully"
			}
			log.Ctx(ctx).Info().Msg(msg)
		}
	}
}
