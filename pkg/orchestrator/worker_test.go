//go:build unit || !integration

package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

const (
	testDequeueTimeout = 5 * time.Millisecond
)

type WorkerTestSuite struct {
	suite.Suite
	schedulerBatchJobs   *MockScheduler
	schedulerServiceJobs *MockScheduler
	schedulerProvider    SchedulerProvider
	broker               *MockEvaluationBroker
	backoff              *backoff.MockBackoff
	worker               *Worker
	eval                 *models.Evaluation
	receiptHandle        string
}

func (s *WorkerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.schedulerBatchJobs = NewMockScheduler(ctrl)
	s.schedulerServiceJobs = NewMockScheduler(ctrl)
	s.broker = NewMockEvaluationBroker(ctrl)
	s.backoff = backoff.NewMockBackoff(ctrl)
	s.schedulerProvider = NewMappedSchedulerProvider(map[string]Scheduler{
		models.JobTypeBatch:   s.schedulerBatchJobs,
		models.JobTypeService: s.schedulerServiceJobs,
	})

	s.eval = mock.Eval()
	s.receiptHandle = uuid.NewString()
	s.worker = NewWorker(WorkerParams{
		SchedulerProvider:     s.schedulerProvider,
		EvaluationBroker:      s.broker,
		dequeueTimeout:        testDequeueTimeout,
		dequeueFailureBackoff: s.backoff,
	})
}

func (s *WorkerTestSuite) TearDownTest() {
	if s.worker != nil && s.worker.Status() != WorkerStatusStopped {
		s.worker.Stop()
	}
}

func (s *WorkerTestSuite) TestProcessEvaluation_Successful() {
	// Mock the dequeueEvaluation method to return a sample evaluation receipt only once
	s.onDequeue().Return(s.eval, s.receiptHandle, nil)
	s.onDequeue().DoAndReturn(s.stopAfterDequeue())

	s.schedulerBatchJobs.EXPECT().Process(context.Background(), s.eval).Return(nil)
	s.broker.EXPECT().Ack(s.eval.ID, s.receiptHandle)

	s.worker.Start(context.Background())
	s.waitUntilStopped()
}

func (s *WorkerTestSuite) TestProcessEvaluation_Shutdown_WhileDequeueing() {
	// Mock the dequeueEvaluation to block until the worker is stopped, and
	// then return a sample evaluation
	s.onDequeue().DoAndReturn(func(schedulers []string, timeout time.Duration) (*models.Evaluation, string, error) {
		// stop while dequeueing
		s.worker.Stop()
		return s.eval, s.receiptHandle, nil
	})

	// The evaluation should be nacked and not processed
	s.broker.EXPECT().Nack(s.eval.ID, s.receiptHandle)

	s.worker.Start(context.Background())
	s.waitUntilStopped()
}

func (s *WorkerTestSuite) TestProcessEvaluation_Shutdown_WhileScheduling() {
	// Mock the dequeueEvaluation method to return a sample evaluation receipt
	s.onDequeue().Return(s.eval, s.receiptHandle, nil)

	// Mock the scheduler's Process method to return no error
	s.schedulerBatchJobs.EXPECT().Process(context.Background(), s.eval).DoAndReturn(func(_ context.Context, _ *models.Evaluation) error {
		// stop while scheduling
		s.worker.Stop()
		return nil
	})

	// The evaluation should be acked even if the worker is stopped
	s.broker.EXPECT().Ack(s.eval.ID, s.receiptHandle)

	s.worker.Start(context.Background())
	s.waitUntilStopped()
}

func (s *WorkerTestSuite) TestProcessEvaluation_Shutdown_ByCancelCtx() {
	s.onDequeue().Return(nil, "", nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	s.worker.Start(ctx)

	// cancel the context should stop the worker
	cancel()
	s.True(s.worker.isShuttingDown(ctx))

	s.Eventually(func() bool {
		return s.worker.Status() == WorkerStatusStopped
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func (s *WorkerTestSuite) TestProcessEvaluation_SchedulerError() {
	// Mock the dequeueEvaluation method to return a sample evaluation receipt
	s.onDequeue().Return(s.eval, s.receiptHandle, nil)
	s.onDequeue().DoAndReturn(s.stopAfterDequeue()) // no more evaluations to dequeue after the first one

	// Mock the scheduler's Process method to return an error
	expectedError := assert.AnError
	s.schedulerBatchJobs.EXPECT().Process(context.Background(), s.eval).Return(expectedError)

	// Expect the evaluationBroker's Nack method to return no error
	s.broker.EXPECT().Nack(s.eval.ID, s.receiptHandle)

	s.worker.Start(context.Background())
	s.waitUntilStopped()
}

func (s *WorkerTestSuite) TestProcessEvaluation_SchedulerMissing() {
	// Mock the dequeueEvaluation method to return a sample evaluation receipt
	s.eval.Type = "missing-scheduler"
	s.onDequeue().Return(s.eval, s.receiptHandle, nil)
	s.onDequeue().DoAndReturn(s.stopAfterDequeue()) // no more evaluations to dequeue after the first one

	// Expect the evaluationBroker's Nack method to return no error
	s.broker.EXPECT().Nack(s.eval.ID, s.receiptHandle)

	s.worker.Start(context.Background())
	s.waitUntilStopped()
}

func (s *WorkerTestSuite) TestProcessEvaluation_BrokerError() {
	// Mock the dequeueEvaluation method to return an error 5 times, and then a sample evaluation receipt
	expectedError := assert.AnError
	s.onDequeue().Return(nil, "", expectedError).Times(5)
	s.onDequeue().DoAndReturn(s.stopAfterDequeue()) // no more evaluations to dequeue

	ctx := context.Background()
	// assert backoff was triggered
	for i := 1; i <= 5; i++ {
		s.backoff.EXPECT().Backoff(ctx, i)
	}

	s.worker.Start(ctx)
	s.waitUntilStopped()
}

func (s *WorkerTestSuite) TestProcessEvaluation_NoEvaluations() {
	// expect the worker to continue polling even after no evaluations are returned
	dequeueCount := 0
	s.onDequeue().DoAndReturn(func(schedulers []string, timeout time.Duration) (*models.Evaluation, string, error) {
		dequeueCount++
		return nil, "", nil
	}).MinTimes(3)

	s.worker.Start(context.Background())

	// wait for at least 3 dequeue calls
	s.Eventually(func() bool {
		return dequeueCount >= 3
	}, 100*time.Millisecond, 10*time.Millisecond)

	// assert still running even after no evaluations are returned
	s.Equal(WorkerStatusRunning, s.worker.Status())
}

func (s *WorkerTestSuite) TestProcessEvaluation_MultiCalls() {
	ctx := context.Background()
	eval1 := mock.Eval()
	eval2 := mock.Eval()
	eval3 := mock.Eval()
	eval4 := mock.Eval()

	eval1.Type = models.JobTypeBatch
	eval2.Type = models.JobTypeService
	eval3.Type = "missing-scheduler"
	eval4.Type = models.JobTypeBatch

	receiptHandle1 := "receiptHandle1"
	receiptHandle2 := "receiptHandle2"
	receiptHandle3 := "receiptHandle3"
	receiptHandle4 := "receiptHandle4"

	expectedError := assert.AnError

	// mock broker to return the evaluations with some failures and empty returns in between
	s.onDequeue().Return(nil, "", expectedError)     // first call to dequeueEvaluation returns an error
	s.onDequeue().Return(eval1, receiptHandle1, nil) // second call returns evaluation
	s.onDequeue().Return(nil, "", nil)               // third call returns no evaluations
	s.onDequeue().Return(eval2, receiptHandle2, nil) // fourth call returns evaluation
	s.onDequeue().Return(nil, "", expectedError)     // fifth call returns an error
	s.onDequeue().Return(nil, "", expectedError)     // sixth call returns an error
	s.onDequeue().Return(eval3, receiptHandle3, nil) // seventh call returns evaluation
	s.onDequeue().Return(eval4, receiptHandle4, nil) // eighth call returns evaluation
	s.onDequeue().DoAndReturn(s.stopAfterDequeue())

	// mock scheduler to fail the first eval, and that scheduler is not called for the third eval
	s.schedulerBatchJobs.EXPECT().Process(context.Background(), eval1).Return(expectedError)
	s.schedulerServiceJobs.EXPECT().Process(context.Background(), eval2)
	s.schedulerBatchJobs.EXPECT().Process(context.Background(), eval4)

	// expect acks for the second and fourth evals
	s.broker.EXPECT().Ack(eval2.ID, receiptHandle2)
	s.broker.EXPECT().Ack(eval4.ID, receiptHandle4)

	// expect broker to nack the first eval due to failure, and third eval due to missing scheduler
	s.broker.EXPECT().Nack(eval1.ID, receiptHandle1)
	s.broker.EXPECT().Nack(eval3.ID, receiptHandle3)

	// assert backoff was triggered as expected
	gomock.InOrder(
		// backoff was triggered for the first error
		s.backoff.EXPECT().Backoff(ctx, 1),
		// back off was reset since evaluations were dequeued after the first error
		s.backoff.EXPECT().Backoff(ctx, 1),
		s.backoff.EXPECT().Backoff(ctx, 2),
	)

	s.worker.Start(ctx)
	s.waitUntilStopped()
}

func (s *WorkerTestSuite) onDequeue() *gomock.Call {
	return s.broker.EXPECT().Dequeue(s.schedulerProvider.EnabledSchedulers(), testDequeueTimeout)
}

func (s *WorkerTestSuite) waitUntilStopped() {
	s.Eventually(func() bool {
		return s.worker.Status() == WorkerStatusStopped
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func (s *WorkerTestSuite) stopAfterDequeue() func(schedulers []string, timeout time.Duration) (*models.Evaluation, string, error) {
	return func(schedulers []string, timeout time.Duration) (*models.Evaluation, string, error) {
		s.worker.Stop()
		return nil, "", nil
	}
}
func TestWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerTestSuite))
}
