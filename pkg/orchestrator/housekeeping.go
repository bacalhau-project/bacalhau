package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	// DefaultHousekeepingWorkers is the default number of parallel workers for housekeeping tasks
	DefaultHousekeepingWorkers = 3
)

type HousekeepingParams struct {
	JobStore jobstore.Store
	// Interval is the interval at which housekeeping tasks are run
	Interval time.Duration
	// Workers is the maximum number of parallel workers for housekeeping tasks
	Workers int
	// TimeoutBuffer is the buffer time to add to the execution timeout
	// It is better that compute nodes timeout and report the failure before the orchestrator does.
	// This buffer is added to the execution timeout to allow for this.
	TimeoutBuffer time.Duration
	// Clock is the clock used for time-based operations.
	// If not provided, the system clock is used.
	Clock clock.Clock
}

type Housekeeping struct {
	jobStore      jobstore.Store
	interval      time.Duration
	timeoutBuffer time.Duration

	workersSem chan struct{}
	waitGroup  sync.WaitGroup
	startOnce  sync.Once
	stopOnce   sync.Once
	stopChan   chan struct{}
	running    bool
	clock      clock.Clock
}

func NewHousekeeping(params HousekeepingParams) (*Housekeeping, error) {
	if params.Workers == 0 {
		params.Workers = DefaultHousekeepingWorkers
	}

	if params.Clock == nil {
		params.Clock = clock.New()
	}

	// validate params
	err := errors.Join(
		validate.IsNotNil(params.JobStore, "job store cannot be nil"),
		validate.IsGreaterThanZero(params.Interval, "interval must be greater than zero"),
		validate.IsGreaterThanZero(params.Workers, "workers must be greater than zero"),
		validate.IsGreaterThanZero(params.TimeoutBuffer, "timeout buffer must be greater than zero"),
	)
	if err != nil {
		return nil, fmt.Errorf("error validating housekeeping params: %w", err)
	}

	h := &Housekeeping{
		jobStore:      params.JobStore,
		interval:      params.Interval,
		timeoutBuffer: params.TimeoutBuffer,
		workersSem:    make(chan struct{}, params.Workers),
		stopChan:      make(chan struct{}),
		clock:         params.Clock,
	}

	return h, nil
}

// IsRunning returns true if the housekeeping task is running
func (h *Housekeeping) IsRunning() bool {
	return h.running
}

// ShouldRun returns true if the housekeeping task should run.
// This is just a placeholder for now until we introduce leader election or lease management for housekeeping
// when we introduce more than one orchestrator.
func (h *Housekeeping) ShouldRun() bool {
	return true
}

// Start starts the housekeeping task
func (h *Housekeeping) Start(ctx context.Context) {
	h.startOnce.Do(func() {
		go h.runHousekeepingTasks(ctx)
	})
}

func (h *Housekeeping) Stop(ctx context.Context) {
	h.stopOnce.Do(func() {
		close(h.stopChan)

		// wait for inflight housekeeping tasks to complete, or until the context is done
		waitGroupDone := make(chan struct{})
		go func() {
			h.waitGroup.Wait()
			close(waitGroupDone)
		}()

		select {
		case <-waitGroupDone:
		case <-ctx.Done():
		}
	})
}

func (h *Housekeeping) runHousekeepingTasks(ctx context.Context) {
	h.running = true
	defer func() { h.running = false }()
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !h.ShouldRun() {
				continue
			}

			// fetch active executions
			activeExecutions := h.fetchActiveExecutions(ctx)

			// run housekeeping tasks
			h.timeoutExecutions(ctx, activeExecutions)
		case <-ctx.Done():
			log.Ctx(ctx).Debug().Msg("Context cancelled, stopping housekeeping task")
			return
		case <-h.stopChan:
			log.Ctx(ctx).Debug().Msg("Stop channel closed, stopping housekeeping task")
			return
		}
	}
}

// fetchActiveExecutions fetches all active executions
func (h *Housekeeping) fetchActiveExecutions(ctx context.Context) []*models.Execution {
	var activeExecutions []*models.Execution
	activeJobs, err := h.jobStore.GetInProgressJobs(ctx, "")
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to get active jobs")
		return activeExecutions
	}

	for i := range activeJobs {
		job := &activeJobs[i]

		// we only have housekeeping tasks for batch and ops jobs
		if job.IsLongRunning() {
			continue
		}

		if h.timeoutJob(ctx, job) {
			continue
		}
		executions, err := h.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
			JobID: job.ID,
		})
		if err != nil {
			// log error and avoid having a single job failure affect the housekeeping of other jobs
			log.Ctx(ctx).Err(err).Msgf("failed to get executions for job %s", job.ID)
			continue
		}
		// filter terminal executions, and enrich executions with job information
		for j := range executions {
			if executions[j].IsTerminalState() {
				continue
			}
			executions[j].Job = job
			activeExecutions = append(activeExecutions, &executions[j])
		}
	}
	return activeExecutions
}

// timeoutJob checks for executions that have been in progress beyond the timeout period
// and enqueue an evaluation for them. It is the responsibility of the scheduler to fail the executions
// returns true if the job was timed out, false otherwise
func (h *Housekeeping) timeoutJob(ctx context.Context, job *models.Job) bool {
	timeoutWithBuffer := job.Task().Timeouts.GetTotalTimeout() + h.timeoutBuffer
	expirationTime := h.clock.Now().Add(-timeoutWithBuffer)
	if job.IsExpired(expirationTime) {
		h.enqueueTimeoutTask(ctx, job, models.EvalTriggerJobTimeout,
			fmt.Sprintf("job %s timed out", job.ID))
		return true
	}
	return false
}

// timeoutExecutions checks for executions that have been in progress beyond the timeout period
// and enqueue an evaluation for them. It is the responsibility of the scheduler to fail the executions
func (h *Housekeeping) timeoutExecutions(ctx context.Context, activeExecutions []*models.Execution) {
	alreadyEvaluatedJobs := make(map[string]struct{})
	for _, execution := range activeExecutions {
		// skip if the job has already been evaluated by another active execution
		if _, ok := alreadyEvaluatedJobs[execution.JobID]; ok {
			continue
		}

		executionTimeout := execution.Job.Task().Timeouts.GetExecutionTimeout()
		if executionTimeout <= 0 {
			continue
		}

		timeoutWithBuffer := executionTimeout + h.timeoutBuffer
		expirationTime := h.clock.Now().Add(-timeoutWithBuffer)
		if execution.IsExpired(expirationTime) {
			alreadyEvaluatedJobs[execution.JobID] = struct{}{}
			h.enqueueTimeoutTask(ctx, execution.Job, models.EvalTriggerExecTimeout,
				fmt.Sprintf("execution %s timed out", execution.ID))
		}
	}
}

func (h *Housekeeping) enqueueTimeoutTask(ctx context.Context, job *models.Job, trigger, comment string) {
	h.workersSem <- struct{}{}
	h.waitGroup.Add(1)

	go func() {
		defer h.waitGroup.Done()
		defer func() { <-h.workersSem }()
		eval := models.NewEvaluation().
			WithJob(job).
			WithTriggeredBy(trigger).
			WithComment(comment).
			Normalize()

		if err := h.jobStore.CreateEvaluation(ctx, *eval); err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to create evaluation %+v", eval)
		} else {
			log.Ctx(ctx).Debug().Msgf("enqueued evaluation for timed-out job/execution %+v", eval)
		}
	}()
}
