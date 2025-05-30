//go:build unit || !integration

package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

const (
	mockJobExecutionTimeout = 60  // 1 minute
	mockJobTotalTimeout     = 120 // 2 minutes

	// another minute. only executions older than 2 minutes are expired, and jobs older than 3 minutes are expired
	timeoutBuffer                  = 1 * time.Minute
	notExpiredModifyTime           = -30 * time.Second
	expiredWithinBufferModifyTime  = -90 * time.Second
	expiredOutsideBufferModifyTime = -120*time.Second - 1*time.Nanosecond

	notExpiredJobCreateTime           = -2 * time.Minute
	expiredJobCreateTimeWithinBuffer  = -3 * time.Minute
	expiredJobCreateTimeOutsideBuffer = -4 * time.Minute
)

type HousekeepingTestSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	clock        *clock.Mock
	mockJobStore *jobstore.MockStore
	housekeeping *Housekeeping
}

func (s *HousekeepingTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.clock = clock.NewMock()
	s.mockJobStore = jobstore.NewMockStore(s.ctrl)

	h, _ := NewHousekeeping(HousekeepingParams{
		JobStore:      s.mockJobStore,
		Interval:      200 * time.Millisecond,
		Workers:       1,
		TimeoutBuffer: timeoutBuffer,
		Clock:         s.clock,
	})

	// we only want to freeze time to have more deterministic tests.
	// It doesn't matter what time it is as we are using relative time to this value
	s.clock.Set(time.Now())
	s.housekeeping = h
}

func (s *HousekeepingTestSuite) TearDownTest() {
	s.housekeeping.Stop(context.Background())
	s.ctrl.Finish()
}

func (s *HousekeepingTestSuite) TestHousekeepingTasks() {
	var tests = []struct {
		name                  string
		JobCreateTime         time.Duration
		ExecutionsModifyTimes []time.Duration
		enqueuedExecTimeouts  int
		enqueuedJobTimeouts   int
		jobCount              int
		jobType               string
		executionState        models.ExecutionStateType
	}{
		{
			name: "ExpiredExecutionOutsideBuffer",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
			},
			jobCount:             1,
			enqueuedExecTimeouts: 1,
		},
		{
			name: "ExpiredExecutionOutsideBufferOpsJobs",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
			},
			jobCount:             1,
			jobType:              models.JobTypeOps,
			enqueuedExecTimeouts: 1,
		},
		{
			name: "ExpiredExecutionWithinBuffer",
			ExecutionsModifyTimes: []time.Duration{
				expiredWithinBufferModifyTime,
			},
			jobCount:             1,
			enqueuedExecTimeouts: 0,
		},
		{
			name: "NoExpiredExecutions",
			ExecutionsModifyTimes: []time.Duration{
				notExpiredModifyTime,
			},
			jobCount:             1,
			enqueuedExecTimeouts: 0,
		},
		{
			name:                "ExpiredJobOutsideBuffer",
			JobCreateTime:       expiredJobCreateTimeOutsideBuffer,
			jobCount:            1,
			enqueuedJobTimeouts: 1,
		},
		{
			name:                "ExpiredJobWithinBuffer",
			JobCreateTime:       expiredJobCreateTimeWithinBuffer,
			jobCount:            1,
			enqueuedJobTimeouts: 0,
		},
		{
			name:                "NoExpiredJo",
			JobCreateTime:       notExpiredJobCreateTime,
			jobCount:            1,
			enqueuedJobTimeouts: 0,
		},
		{
			name: "MultipleExecutionTimeout",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
				expiredOutsideBufferModifyTime,
			},
			jobCount:             1,
			enqueuedExecTimeouts: 1,
		},
		{
			name: "JobExpirationHasHigherPrecedence",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
				expiredOutsideBufferModifyTime,
			},
			JobCreateTime:       expiredJobCreateTimeOutsideBuffer,
			jobCount:            1,
			enqueuedJobTimeouts: 1,
		},
		{
			name: "NoopOnTerminalExecutions",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
			},
			executionState:       models.ExecutionStateCompleted,
			jobCount:             1,
			enqueuedExecTimeouts: 0,
		},
		{
			name: "NoopOnServiceJobs",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
			},
			jobCount:             1,
			jobType:              models.JobTypeService,
			enqueuedExecTimeouts: 0,
		},
		{
			name: "NoopOnDaemonJobs",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
			},
			jobCount:             1,
			jobType:              models.JobTypeDaemon,
			enqueuedExecTimeouts: 0,
		},
		{
			name: "MultipleJobsExecutionTimeout",
			ExecutionsModifyTimes: []time.Duration{
				expiredOutsideBufferModifyTime,
			},
			jobCount:             2,
			enqueuedExecTimeouts: 2,
		},
		{
			name: "MultipleJobsMultipleExecutionTimeout",
			ExecutionsModifyTimes: []time.Duration{
				notExpiredModifyTime,
				expiredWithinBufferModifyTime,
				expiredOutsideBufferModifyTime,
				expiredOutsideBufferModifyTime,
			},
			jobCount:             2,
			enqueuedExecTimeouts: 2,
		},
		{
			name:                 "TestNoActiveJobsFound",
			jobCount:             0,
			enqueuedExecTimeouts: 0,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			jobs := make([]models.Job, tc.jobCount)
			jobsMap := make(map[string][]models.Execution)

			// prepare jobs and executions mock data
			for i := 0; i < tc.jobCount; i++ {
				job, executions := s.mockJob(tc.JobCreateTime, tc.ExecutionsModifyTimes...)

				// set execution state if provided
				for j := range executions {
					if tc.executionState.IsUndefined() {
						executions[j].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
					} else {
						executions[j].ComputeState = models.NewExecutionState(tc.executionState)
					}
				}

				// set job type if provided
				if tc.jobType != "" {
					job.Type = tc.jobType
				} else {
					job.Type = models.JobTypeBatch
				}

				jobs[i] = *job
				jobsMap[job.ID] = executions
			}

			// mock job store calls
			s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").Times(1).Return(jobs, nil)
			s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").AnyTimes().Return([]models.Job{}, nil)
			for _, job := range jobs {
				if job.IsLongRunning() {
					continue
				}
				if tc.enqueuedJobTimeouts > 0 {
					continue
				}
				s.mockJobStore.EXPECT().GetExecutions(
					gomock.Any(),
					jobstore.GetExecutionsOptions{
						JobID:                   job.ID,
						CurrentLatestJobVersion: job.Version,
					},
				).Return(jobsMap[job.ID], nil)
			}

			// assert evaluation enqueued for each job
			for i := 0; i < tc.enqueuedExecTimeouts; i++ {
				s.assertEvaluationEnqueued(jobs[i], models.EvalTriggerExecTimeout)
			}
			for i := 0; i < tc.enqueuedJobTimeouts; i++ {
				s.assertEvaluationEnqueued(jobs[i], models.EvalTriggerJobTimeout)
			}

			s.housekeeping.Start(context.Background())
			s.Eventually(func() bool { return s.ctrl.Satisfied() }, 3*time.Second, 50*time.Millisecond)

			// if no-op is expected, wait another 200ms to ensure that
			if (tc.enqueuedExecTimeouts + tc.enqueuedJobTimeouts) == 0 {
				time.Sleep(200 * time.Millisecond)
			}

			s.TearDownTest()
		})
	}
}

// TestMultipleHousekeepingRounds tests that housekeeping tasks are run multiple times
// and in each time we are picking new set of jobs
func (s *HousekeepingTestSuite) TestMultipleHousekeepingRounds() {
	job1, executions1 := s.mockJob(notExpiredJobCreateTime, expiredOutsideBufferModifyTime)
	job2, executions2 := s.mockJob(expiredJobCreateTimeWithinBuffer, expiredOutsideBufferModifyTime)
	job3, _ := s.mockJob(expiredJobCreateTimeOutsideBuffer, expiredOutsideBufferModifyTime)

	// mock job store calls where we return a single job in each call
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").Times(1).Return([]models.Job{*job1}, nil)
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").Times(1).Return([]models.Job{*job2}, nil)
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").Times(1).Return([]models.Job{*job3}, nil)
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").AnyTimes().Return([]models.Job{}, nil)

	// mock executions call for each job
	s.mockJobStore.EXPECT().GetExecutions(
		gomock.Any(),
		jobstore.GetExecutionsOptions{
			JobID:                   job1.ID,
			CurrentLatestJobVersion: job1.Version,
		},
	).Return(executions1, nil)
	s.mockJobStore.EXPECT().GetExecutions(
		gomock.Any(),
		jobstore.GetExecutionsOptions{
			JobID:                   job2.ID,
			CurrentLatestJobVersion: job2.Version,
		},
	).Return(executions2, nil)

	// assert evaluation enqueued for each job
	s.assertEvaluationEnqueued(*job1, models.EvalTriggerExecTimeout)
	s.assertEvaluationEnqueued(*job2, models.EvalTriggerExecTimeout)
	s.assertEvaluationEnqueued(*job3, models.EvalTriggerJobTimeout)

	s.housekeeping.Start(context.Background())
	s.Eventually(func() bool { return s.ctrl.Satisfied() }, 3*time.Second, 50*time.Millisecond)
}

func (s *HousekeepingTestSuite) TestShouldRun() {
	s.True(s.housekeeping.ShouldRun())
}

func (s *HousekeepingTestSuite) TestStop() {
	s.housekeeping.Start(context.Background())
	s.Eventually(func() bool { return s.housekeeping.IsRunning() }, 1*time.Second, 10*time.Millisecond)

	s.housekeeping.Stop(context.Background())

	select {
	case <-s.housekeeping.stopChan:
		s.True(true)
	default:
		s.Fail("context should be cancelled")
	}

	s.Eventuallyf(func() bool { return !s.housekeeping.IsRunning() }, 1*time.Second, 10*time.Millisecond,
		"Housekeeping should not be running after Stop")
}

func (s *HousekeepingTestSuite) TestStopWithoutStart() {
	s.housekeeping.Stop(context.Background()) // Expect no panic or error

	select {
	case <-s.housekeeping.stopChan:
		s.True(true, "stopChan should be closed")
	default:
		s.Fail("stopChan should be closed after Stop is called")
	}

	s.False(s.housekeeping.IsRunning())
}

func (s *HousekeepingTestSuite) TestStartMultipleTimes() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.housekeeping.Start(ctx)
	s.housekeeping.Start(ctx) // no panics
}

func (s *HousekeepingTestSuite) TestStopMultipleTimes() {
	s.housekeeping.Start(context.Background())
	s.housekeeping.Stop(context.Background())
	s.housekeeping.Stop(context.Background()) // Second call should gracefully do nothing

	select {
	case <-s.housekeeping.stopChan:
		s.True(true, "stopChan should be closed")
	default:
		s.Fail("stopChan should remain closed after multiple stops")
	}

	s.Eventuallyf(func() bool { return !s.housekeeping.IsRunning() }, 1*time.Second, 10*time.Millisecond,
		"Housekeeping should not be running after Stop")
}

func (s *HousekeepingTestSuite) TestStopRespectsContext() {
	s.housekeeping.Start(context.Background())

	// Create a context that will be cancelled after 500 milliseconds
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Simulate a long-running task by adding to the WaitGroup and not marking it as done
	s.housekeeping.waitGroup.Add(1)

	// Call Stop with the context that will be cancelled
	go s.housekeeping.Stop(ctx)

	select {
	case <-ctx.Done():
		// If the context is done, it means that Stop respected the context and returned early
		s.True(true, "Stop should return early if the context is done")
	case <-time.After(1 * time.Second):
		// If we reach this case, it means that Stop did not respect the context and did not return early
		s.Fail("Stop did not return early even though the context was done")
	}

	s.Eventuallyf(func() bool { return !s.housekeeping.IsRunning() }, 1*time.Second, 10*time.Millisecond,
		"Housekeeping should not be running after Stop")
}

func (s *HousekeepingTestSuite) TestCancelProvidedContext() {
	ctx, cancel := context.WithCancel(context.Background())

	s.housekeeping.Start(ctx)
	// Eventually the housekeeping should be running
	s.Eventually(func() bool { return s.housekeeping.IsRunning() }, 1*time.Second, 10*time.Millisecond)

	cancel() // Cancel the context
	s.Eventually(func() bool { return !s.housekeeping.IsRunning() }, 1*time.Second, 10*time.Millisecond)
}

func (s *HousekeepingTestSuite) assertEvaluationEnqueued(job models.Job, trigger string) {
	matchEvaluation := func(eval *models.Evaluation) {
		s.Require().Equal(job.ID, eval.JobID)
		s.Require().Equal(trigger, eval.TriggeredBy)
		s.Require().Equal(job.Type, eval.Type)
	}
	s.mockJobStore.EXPECT().CreateEvaluation(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, eval models.Evaluation) {
		matchEvaluation(&eval)
	}).Return(nil)
}

// mockJob creates a mock job for testing. It takes list of ModifyTime for executions
func (s *HousekeepingTestSuite) mockJob(CreateTime time.Duration, ModifyTime ...time.Duration) (*models.Job, []models.Execution) {
	job := mock.Job()
	job.CreateTime = s.clock.Now().Add(CreateTime).UnixNano()
	job.Task().Timeouts.ExecutionTimeout = mockJobExecutionTimeout
	job.Task().Timeouts.TotalTimeout = mockJobTotalTimeout

	executions := make([]models.Execution, 0, len(ModifyTime))
	for _, t := range ModifyTime {
		execution := mock.ExecutionForJob(job)
		execution.ModifyTime = s.clock.Now().Add(t).UnixNano()
		execution.ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
		executions = append(executions, *execution)
	}
	return job, executions
}

func TestHousekeepingTestSuite(t *testing.T) {
	suite.Run(t, new(HousekeepingTestSuite))
}
