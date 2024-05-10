//go:build unit || !integration

package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockJobTimeout                 = 60              // 1 minute
	timeoutBuffer                  = 1 * time.Minute // another minute. only executions older than 2 minutes are expired
	notExpiredModifyTime           = -30 * time.Second
	expiredWithinBufferModifyTime  = -90 * time.Second
	expiredOutsideBufferModifyTime = -120 * time.Second
)

type HousekeepingTestSuite struct {
	suite.Suite
	ctrl                 *gomock.Controller
	mockJobStore         *jobstore.MockStore
	mockEvaluationBroker *MockEvaluationBroker
	housekeeping         *Housekeeping
}

func (s *HousekeepingTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockJobStore = jobstore.NewMockStore(s.ctrl)
	s.mockEvaluationBroker = NewMockEvaluationBroker(s.ctrl)

	h, _ := NewHousekeeping(HousekeepingParams{
		EvaluationBroker: s.mockEvaluationBroker,
		JobStore:         s.mockJobStore,
		Interval:         200 * time.Millisecond,
		Workers:          1,
		TimeoutBuffer:    timeoutBuffer,
	})

	s.housekeeping = h
}

func (s *HousekeepingTestSuite) TearDownTest() {
	s.housekeeping.Stop(context.Background())
	s.ctrl.Finish()
}

func (s *HousekeepingTestSuite) TestHousekeepingTasks() {
	var tests = []struct {
		name                 string
		ModifyTimes          []time.Time
		expectedEnqueueCount int
		jobCount             int
		jobType              string
		executionState       models.ExecutionStateType
	}{
		{
			name: "ExpiredExecutionOutsideBuffer",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			jobCount:             1,
			expectedEnqueueCount: 1,
		},
		{
			name: "ExpiredExecutionOutsideBufferOpsJobs",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			jobCount:             1,
			jobType:              models.JobTypeOps,
			expectedEnqueueCount: 1,
		},
		{
			name: "ExpiredExecutionWithinBuffer",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredWithinBufferModifyTime),
			},
			jobCount:             1,
			expectedEnqueueCount: 0,
		},
		{
			name: "NoExpiredExecutions",
			ModifyTimes: []time.Time{
				time.Now().Add(notExpiredModifyTime),
			},
			jobCount:             1,
			expectedEnqueueCount: 0,
		},
		{
			name: "MultipleExecutionTimeout",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredOutsideBufferModifyTime),
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			jobCount:             1,
			expectedEnqueueCount: 1,
		},

		{
			name: "NoopOnTerminalExecutions",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			executionState:       models.ExecutionStateCompleted,
			jobCount:             1,
			expectedEnqueueCount: 0,
		},
		{
			name: "NoopOnServiceJobs",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			jobCount:             1,
			jobType:              models.JobTypeService,
			expectedEnqueueCount: 0,
		},
		{
			name: "NoopOnDaemonJobs",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			jobCount:             1,
			jobType:              models.JobTypeDaemon,
			expectedEnqueueCount: 0,
		},
		{
			name: "MultipleJobsExecutionTimeout",
			ModifyTimes: []time.Time{
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			jobCount:             2,
			expectedEnqueueCount: 2,
		},
		{
			name: "MultipleJobsMultipleExecutionTimeout",
			ModifyTimes: []time.Time{
				time.Now().Add(notExpiredModifyTime),
				time.Now().Add(expiredWithinBufferModifyTime),
				time.Now().Add(expiredOutsideBufferModifyTime),
				time.Now().Add(expiredOutsideBufferModifyTime),
			},
			jobCount:             2,
			expectedEnqueueCount: 2,
		},
		{
			name:                 "TestNoActiveJobsFound",
			jobCount:             0,
			expectedEnqueueCount: 0,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			jobs := make([]models.Job, tc.jobCount)
			jobsMap := make(map[string][]models.Execution)

			// prepare jobs and executions mock data
			for i := 0; i < tc.jobCount; i++ {
				job, executions := mockJob(tc.ModifyTimes...)

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
				s.mockJobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(jobsMap[job.ID], nil)
			}

			// assert evaluation enqueued for each job
			for i := 0; i < tc.expectedEnqueueCount; i++ {
				s.assertEvaluationEnqueued(jobs[i])
			}

			s.housekeeping.Start(context.Background())
			s.Eventually(func() bool { return s.ctrl.Satisfied() }, 3*time.Second, 50*time.Millisecond)

			// if no-op is expected, wait another 200ms to ensure that
			if tc.expectedEnqueueCount == 0 {
				time.Sleep(200 * time.Millisecond)
			}

			s.TearDownTest()
		})
	}
}

// TestMultipleHousekeepingRounds tests that housekeeping tasks are run multiple times
// and in each time we are picking new set of jobs
func (s *HousekeepingTestSuite) TestMultipleHousekeepingRounds() {
	job1, executions1 := mockJob(time.Now().Add(expiredOutsideBufferModifyTime))
	job2, executions2 := mockJob(time.Now().Add(expiredOutsideBufferModifyTime))
	job3, executions3 := mockJob(time.Now().Add(expiredOutsideBufferModifyTime))

	// mock job store calls where we return a single job in each call
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").Times(1).Return([]models.Job{*job1}, nil)
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").Times(1).Return([]models.Job{*job2}, nil)
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").Times(1).Return([]models.Job{*job3}, nil)
	s.mockJobStore.EXPECT().GetInProgressJobs(gomock.Any(), "").AnyTimes().Return([]models.Job{}, nil)

	// mock executions call for each job
	s.mockJobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job1.ID}).Return(executions1, nil)
	s.mockJobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job2.ID}).Return(executions2, nil)
	s.mockJobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job3.ID}).Return(executions3, nil)

	// assert evaluation enqueued for each job
	s.assertEvaluationEnqueued(*job1)
	s.assertEvaluationEnqueued(*job2)
	s.assertEvaluationEnqueued(*job3)

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

func (s *HousekeepingTestSuite) assertEvaluationEnqueued(job models.Job) {
	matchEvaluation := func(eval *models.Evaluation) {
		s.Require().Equal(job.ID, eval.JobID)
		s.Require().Equal(models.EvalTriggerExecTimeout, eval.TriggeredBy)
		s.Require().Equal(job.Type, eval.Type)
	}
	s.mockJobStore.EXPECT().CreateEvaluation(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, eval models.Evaluation) {
		matchEvaluation(&eval)
	}).Return(nil)

	s.mockEvaluationBroker.EXPECT().Enqueue(gomock.Any()).Do(func(eval *models.Evaluation) {
		matchEvaluation(eval)
	}).Return(nil)
}

// mockJob creates a mock job for testing. It takes list of ModifyTime for executions
func mockJob(ModifyTime ...time.Time) (*models.Job, []models.Execution) {
	job := mock.Job()
	job.Task().Timeouts.ExecutionTimeout = mockJobTimeout

	executions := make([]models.Execution, 0, len(ModifyTime))
	for _, t := range ModifyTime {
		execution := mock.ExecutionForJob(job)
		execution.ModifyTime = t.UnixNano()
		execution.ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
		executions = append(executions, *execution)
	}
	return job, executions
}

func TestHousekeepingTestSuite(t *testing.T) {
	suite.Run(t, new(HousekeepingTestSuite))
}
