//go:build unit || !integration

package nodes

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

type ReEvaluatorTestSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	clock        *clock.Mock
	mockJobStore *jobstore.MockStore
	reEvaluator  *ReEvaluator
}

func (s *ReEvaluatorTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.clock = clock.NewMock()
	s.mockJobStore = jobstore.NewMockStore(s.ctrl)

	re, err := NewReEvaluator(ReEvaluatorParams{
		JobStore:         s.mockJobStore,
		BatchDelay:       100 * time.Millisecond,
		MaxBatchSize:     10,
		EventChannelSize: 100,
		Clock:            s.clock,
	})
	s.Require().NoError(err)

	s.reEvaluator = re
}

func (s *ReEvaluatorTestSuite) TearDownTest() {
	if s.reEvaluator != nil {
		s.reEvaluator.Stop(context.Background())
	}
	s.ctrl.Finish()
}

func (s *ReEvaluatorTestSuite) TestNewNodeReEvaluator() {
	// Test successful creation
	re, err := NewReEvaluator(ReEvaluatorParams{
		JobStore: s.mockJobStore,
	})
	s.NoError(err)
	s.NotNil(re)
	s.Equal(defaultBatchDelay, re.batchDelay)
	s.Equal(defaultMaxBatchSize, re.maxBatchSize)
	s.Equal(defaultEventChannelSize, cap(re.eventChan))

	// Test validation errors
	_, err = NewReEvaluator(ReEvaluatorParams{
		JobStore: nil,
	})
	s.Error(err)
}

func (s *ReEvaluatorTestSuite) TestStartStop() {
	// Test start - should always start successfully
	err := s.reEvaluator.Start(context.Background())
	s.NoError(err)
	s.True(s.reEvaluator.IsRunning())

	// Test stop
	err = s.reEvaluator.Stop(context.Background())
	s.NoError(err)
	s.False(s.reEvaluator.IsRunning())
}

func (s *ReEvaluatorTestSuite) TestNodeJoinEvent() {
	s.setupReEvaluator()

	nodeID := "test-node-1"

	// Create test jobs
	daemonJob := s.createTestJob(models.JobTypeDaemon)
	batchJob := s.createTestJob(models.JobTypeBatch)

	// Mock daemon jobs query
	s.mockJobStore.EXPECT().
		GetInProgressJobs(gomock.Any(), models.JobTypeDaemon).
		Return([]models.Job{*daemonJob}, nil)

	// Mock executions query for the joining node
	execution := s.createTestExecution(batchJob.ID, nodeID)
	s.expectExecutionsQuery([]string{nodeID}, []models.Execution{*execution})

	// Expect evaluations to be created (order may vary due to map iteration)
	s.mockJobStore.EXPECT().
		CreateEvaluation(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, eval models.Evaluation) {
			// Should be either daemonJob or batchJob
			s.Equal(models.EvalTriggerNodeJoin, eval.TriggeredBy)
			if eval.JobID == daemonJob.ID {
				s.Equal(daemonJob.Type, eval.Type)
			} else {
				s.Equal(batchJob.Type, eval.Type)
			}
		}).
		Return(nil).
		Times(2)

	// Trigger node join event and wait for processing
	event := s.createNodeEvent(nodeID, models.NodeStates.DISCONNECTED, models.NodeStates.CONNECTED)
	s.reEvaluator.HandleNodeConnectionEvent(event)
	s.triggerProcessing()

	// Wait for processing to complete
	s.waitForCompletion()
}

func (s *ReEvaluatorTestSuite) TestNodeLeaveEvent() {
	s.setupReEvaluator()

	nodeID := "test-node-1"

	// Create test job
	batchJob := s.createTestJob(models.JobTypeBatch)

	// Mock executions query for the leaving node
	execution := s.createTestExecution(batchJob.ID, nodeID)
	s.expectExecutionsQuery([]string{nodeID}, []models.Execution{*execution})

	// Expect evaluation to be created
	s.expectEvaluationCreated(batchJob, models.EvalTriggerNodeLeave)

	// Trigger node leave event and wait for processing
	event := s.createNodeEvent(nodeID, models.NodeStates.CONNECTED, models.NodeStates.DISCONNECTED)
	s.reEvaluator.HandleNodeConnectionEvent(event)
	s.triggerProcessing()

	// Wait for processing to complete
	s.waitForCompletion()
}

func (s *ReEvaluatorTestSuite) TestBatchingSameEventType() {
	s.setupReEvaluator()

	nodeIDs := []string{"node-1", "node-2", "node-3"}

	// Create test job
	batchJob := s.createTestJob(models.JobTypeBatch)

	// Mock executions query for all nodes - expect one batched call with all node IDs
	var allExecutions []models.Execution
	for _, nodeID := range nodeIDs {
		execution := s.createTestExecution(batchJob.ID, nodeID)
		allExecutions = append(allExecutions, *execution)
	}
	// Mock executions query with flexible node ID matching
	s.expectExecutionsQuery(nodeIDs, allExecutions)

	// Expect only one evaluation due to deduplication
	s.expectEvaluationCreated(batchJob, models.EvalTriggerNodeLeave)

	// Trigger multiple leave events
	for _, nodeID := range nodeIDs {
		event := s.createNodeEvent(nodeID, models.NodeStates.CONNECTED, models.NodeStates.DISCONNECTED)
		s.reEvaluator.HandleNodeConnectionEvent(event)
	}
	s.triggerProcessing()

	// Wait for processing to complete
	s.waitForCompletion()
}

func (s *ReEvaluatorTestSuite) TestMaxBatchSizeLimit() {
	s.setupReEvaluator()

	// Create more events than the batch size limit
	numNodes := s.reEvaluator.maxBatchSize + 5
	nodeIDs := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		nodeIDs[i] = "node-" + string(rune('0'+i))
	}

	// Mock batched responses - expect batched calls but with flexible node ID order
	s.mockJobStore.EXPECT().
		GetExecutions(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, opts jobstore.GetExecutionsOptions) ([]models.Execution, error) {
			// Should have InProgressOnly=true and IncludeJob=true
			s.True(opts.InProgressOnly)
			s.True(opts.IncludeJob)
			// Should batch some nodes together (at least more than 1, up to maxBatchSize)
			s.True(len(opts.NodeIDs) >= 1 && len(opts.NodeIDs) <= s.reEvaluator.maxBatchSize)
			return []models.Execution{}, nil
		}).
		AnyTimes()

	// Note: processBatchedEvents is not exposed, so we'll rely on the timing mechanism
	// to ensure batching works correctly

	// Trigger events
	for _, nodeID := range nodeIDs {
		event := NodeConnectionEvent{
			NodeID:    nodeID,
			Previous:  models.NodeStates.CONNECTED,
			Current:   models.NodeStates.DISCONNECTED,
			Timestamp: s.clock.Now(),
		}
		s.reEvaluator.HandleNodeConnectionEvent(event)
	}

	s.triggerProcessing()

	// Wait for processing to complete
	s.waitForCompletion()
}

func (s *ReEvaluatorTestSuite) TestIgnoreNonConnectionStateChanges() {
	s.setupReEvaluator()

	// Event that doesn't involve connection/disconnection
	event := NodeConnectionEvent{
		NodeID:    "test-node",
		Previous:  models.NodeStates.DISCONNECTED,
		Current:   models.NodeStates.DISCONNECTED,
		Timestamp: s.clock.Now(),
	}

	// Should not process this event
	s.reEvaluator.HandleNodeConnectionEvent(event)
	// Trigger processing - should not create any evaluations
	s.triggerProcessing()
}

// Helper methods

func (s *ReEvaluatorTestSuite) setupReEvaluator() {
	err := s.reEvaluator.Start(context.Background())
	s.Require().NoError(err)
}

func (s *ReEvaluatorTestSuite) createTestJob(jobType string) *models.Job {
	job := mock.Job()
	job.Type = jobType
	job.State = models.NewJobState(models.JobStateTypeRunning)
	return job
}

func (s *ReEvaluatorTestSuite) createTestExecution(jobID, nodeID string) *models.Execution {
	execution := mock.Execution()
	execution.JobID = jobID
	execution.NodeID = nodeID
	execution.ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	execution.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
	return execution
}

func (s *ReEvaluatorTestSuite) expectEvaluationCreated(job *models.Job, trigger string) {
	s.mockJobStore.EXPECT().
		CreateEvaluation(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, eval models.Evaluation) {
			s.Equal(job.ID, eval.JobID)
			s.Equal(trigger, eval.TriggeredBy)
			s.Equal(job.Type, eval.Type)
		}).
		Return(nil)
}

// Helper methods for improved test readability

func (s *ReEvaluatorTestSuite) createNodeEvent(nodeID string, previous, current models.NodeConnectionState) NodeConnectionEvent {
	return NodeConnectionEvent{
		NodeID:    nodeID,
		Previous:  previous,
		Current:   current,
		Timestamp: s.clock.Now(),
	}
}

func (s *ReEvaluatorTestSuite) triggerProcessing() {
	// wait for the channel to be processed and empty
	s.Eventually(func() bool {
		return len(s.reEvaluator.eventChan) == 0
	}, time.Second, 10*time.Millisecond)

	// Wait a little longer for the last event to be batched
	time.Sleep(20 * time.Millisecond)

	// Advance clock to trigger batch processing
	s.clock.Add(s.reEvaluator.batchDelay + time.Millisecond)
}

func (s *ReEvaluatorTestSuite) expectExecutionsQuery(nodeIDs []string, executions []models.Execution) {
	s.mockJobStore.EXPECT().
		GetExecutions(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, options jobstore.GetExecutionsOptions) ([]models.Execution, error) {
			s.ElementsMatch(options.NodeIDs, nodeIDs)
			s.True(options.InProgressOnly)
			s.True(options.IncludeJob)
			return executions, nil
		})
}

func (s *ReEvaluatorTestSuite) waitForCompletion() {
	s.Eventually(func() bool {
		return s.ctrl.Satisfied()
	}, time.Second, 10*time.Millisecond)
}

func TestNodeReEvaluatorTestSuite(t *testing.T) {
	suite.Run(t, new(ReEvaluatorTestSuite))
}
