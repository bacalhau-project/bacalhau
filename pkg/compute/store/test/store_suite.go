package test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type StoreCreator func(ctx context.Context, dbPath string) (store.ExecutionStore, error)

type StoreSuite struct {
	suite.Suite
	ctx            context.Context
	executionStore store.ExecutionStore
	execution      *models.Execution
	storeCreator   StoreCreator
	dbPath         string
}

func (s *StoreSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
}

func (s *StoreSuite) SetupTest() {
	var err error
	s.ctx = context.Background()
	s.dbPath = s.T().TempDir()
	s.executionStore, err = s.storeCreator(s.ctx, s.dbPath)
	require.NoError(s.T(), err)
	s.execution = mock.ExecutionForJob(mock.Job())
}

func (s *StoreSuite) TearDownTest() {
	if s.executionStore != nil {
		s.NoError(s.executionStore.Close(s.ctx))
	}
	_ = os.Remove(s.dbPath)
}

func RunStoreSuite(t *testing.T, creator StoreCreator) {
	s := new(StoreSuite)
	s.storeCreator = creator
	suite.Run(t, s)
}

func (s *StoreSuite) TestCreateExecution() {
	events := []*models.Event{{Topic: "TestEvent"}}
	err := s.executionStore.CreateExecution(s.ctx, *s.execution, events...)
	s.Require().NoError(err)

	// verify the execution was created
	readExecution, err := s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Equal(s.execution.ID, readExecution.ID)

	// verify events were created
	readEvents, err := s.executionStore.GetExecutionEvents(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Len(readEvents, 1)
	s.Equal(events[0].Topic, readEvents[0].Topic)

	// verify create event was created in the event store
	eventsResp, err := s.executionStore.GetEventStore().GetEvents(s.ctx, watcher.GetEventsRequest{
		EventIterator: watcher.TrimHorizonIterator(),
		Filter:        watcher.EventFilter{ObjectTypes: []string{compute.EventObjectExecutionUpsert}},
	})
	s.Require().NoError(err)
	s.Len(eventsResp.Events, 1)
	s.verifyWatcherExecutionEvent(eventsResp.Events[0], watcher.OperationCreate, readExecution, models.Execution{})
}

func (s *StoreSuite) TestUpdateExecution() {
	err := s.executionStore.CreateExecution(s.ctx, *s.execution)
	s.Require().NoError(err)

	createdExecution, err := s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)

	// update with new state
	newState := models.ExecutionStatePublishing
	updateRequest := store.UpdateExecutionRequest{
		ExecutionID: s.execution.ID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(newState),
		},
		Events: []*models.Event{{Topic: "UpdateEvent"}},
	}
	err = s.executionStore.UpdateExecutionState(s.ctx, updateRequest)
	s.Require().NoError(err)

	// verify the update happened as expected
	updatedExecution, err := s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Equal(newState, updatedExecution.ComputeState.StateType)
	s.Equal(createdExecution.Revision+1, updatedExecution.Revision)

	// verify events were updated
	events, err := s.executionStore.GetExecutionEvents(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Len(events, 1)
	s.Equal("UpdateEvent", string(events[0].Topic))

	// verify update event was created in the event store
	eventsResp, err := s.executionStore.GetEventStore().GetEvents(s.ctx, watcher.GetEventsRequest{
		EventIterator: watcher.TrimHorizonIterator(),
		Filter:        watcher.EventFilter{ObjectTypes: []string{compute.EventObjectExecutionUpsert}},
	})
	s.Require().NoError(err)
	s.Len(eventsResp.Events, 2)
	s.verifyWatcherExecutionEvent(eventsResp.Events[0], watcher.OperationCreate, createdExecution, models.Execution{})
	s.verifyWatcherExecutionEvent(eventsResp.Events[1], watcher.OperationUpdate, updatedExecution, *createdExecution)
}

func (s *StoreSuite) TestGetExecutionCount() {
	states := []models.ExecutionStateType{
		models.ExecutionStateBidAccepted,
		models.ExecutionStateBidAccepted,
		models.ExecutionStatePublishing,
		models.ExecutionStateCompleted,
		models.ExecutionStateCompleted,
	}

	for _, state := range states {
		execution := mock.ExecutionForJob(mock.Job())
		err := s.executionStore.CreateExecution(s.ctx, *execution)
		s.Require().NoError(err)

		updateRequest := store.UpdateExecutionRequest{
			ExecutionID: execution.ID,
			NewValues: models.Execution{
				ComputeState: models.NewExecutionState(state),
			},
		}

		err = s.executionStore.UpdateExecutionState(s.ctx, updateRequest)
		s.Require().NoError(err)
	}

	// Test GetExecutionCount
	bidAcceptedCount, err := s.executionStore.GetExecutionCount(s.ctx, models.ExecutionStateBidAccepted)
	s.Require().NoError(err)
	s.Equal(uint64(2), bidAcceptedCount)

	publishingCount, err := s.executionStore.GetExecutionCount(s.ctx, models.ExecutionStatePublishing)
	s.Require().NoError(err)
	s.Equal(uint64(1), publishingCount)

	completedCount, err := s.executionStore.GetExecutionCount(s.ctx, models.ExecutionStateCompleted)
	s.Require().NoError(err)
	s.Equal(uint64(2), completedCount)
}

func (s *StoreSuite) TestDeleteExecutionDoesntExist() {
	err := s.executionStore.DeleteExecution(s.ctx, uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
}

func (s *StoreSuite) TestGetExecutionEventsDoesntExist() {
	_, err := s.executionStore.GetExecutionEvents(s.ctx, uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionEventsNotFound{})
}

func (s *StoreSuite) TestGetCheckpointReturnsZeroForNonExistent() {
	seq, err := s.executionStore.GetCheckpoint(s.ctx, "test-checkpoint")
	s.Require().NoError(err)
	s.Equal(uint64(0), seq)
}

func (s *StoreSuite) TestCheckpointStoresAndRetrieves() {
	checkpointName := "test-checkpoint"
	expectedSeq := uint64(42)

	// Store checkpoint
	err := s.executionStore.Checkpoint(s.ctx, checkpointName, expectedSeq)
	s.Require().NoError(err)

	// Retrieve checkpoint
	seq, err := s.executionStore.GetCheckpoint(s.ctx, checkpointName)
	s.Require().NoError(err)
	s.Equal(expectedSeq, seq)
}

func (s *StoreSuite) TestMultipleCheckpointsCoexist() {
	checkpoints := map[string]uint64{
		"checkpoint-1": 42,
		"checkpoint-2": 100,
		"checkpoint-3": 999,
	}

	// Store multiple checkpoints
	for name, seq := range checkpoints {
		err := s.executionStore.Checkpoint(s.ctx, name, seq)
		s.Require().NoError(err)
	}

	// Verify each checkpoint
	for name, expectedSeq := range checkpoints {
		seq, err := s.executionStore.GetCheckpoint(s.ctx, name)
		s.Require().NoError(err)
		s.Equal(expectedSeq, seq)
	}
}

func (s *StoreSuite) TestCheckpointCanBeUpdated() {
	checkpointName := "update-test"
	initialSeq := uint64(42)
	updatedSeq := uint64(100)

	// Store initial checkpoint
	err := s.executionStore.Checkpoint(s.ctx, checkpointName, initialSeq)
	s.Require().NoError(err)

	// Verify initial value
	seq, err := s.executionStore.GetCheckpoint(s.ctx, checkpointName)
	s.Require().NoError(err)
	s.Equal(initialSeq, seq)

	// Update checkpoint
	err = s.executionStore.Checkpoint(s.ctx, checkpointName, updatedSeq)
	s.Require().NoError(err)

	// Verify updated value
	seq, err = s.executionStore.GetCheckpoint(s.ctx, checkpointName)
	s.Require().NoError(err)
	s.Equal(updatedSeq, seq)
}

func (s *StoreSuite) TestCheckpointWithClosedStore() {
	s.Require().NoError(s.executionStore.Close(s.ctx))

	// Try to set checkpoint after closing
	err := s.executionStore.Checkpoint(s.ctx, "test-checkpoint", 42)
	s.Error(err)

	// Try to get checkpoint after closing
	_, err = s.executionStore.GetCheckpoint(s.ctx, "test-checkpoint")
	s.Error(err)
}

func (s *StoreSuite) TestGetCheckpointBlankName() {
	_, err := s.executionStore.GetCheckpoint(s.ctx, "")
	s.ErrorAs(err, &store.ErrCheckpointNameBlank{})
}

func (s *StoreSuite) TestCheckpointBlankName() {
	err := s.executionStore.Checkpoint(s.ctx, "", 42)
	s.ErrorAs(err, &store.ErrCheckpointNameBlank{})
}

func (s *StoreSuite) verifyWatcherExecutionEvent(event watcher.Event,
	expectedOperation watcher.Operation,
	execution *models.Execution,
	previousExecution models.Execution) {
	s.Equal(expectedOperation, event.Operation)
	executionUpsert, ok := event.Object.(models.ExecutionUpsert)
	s.True(ok)
	s.Equal(execution, executionUpsert.Current)

	if expectedOperation == watcher.OperationCreate {
		s.Nil(executionUpsert.Previous)
	} else {
		s.Equal(&previousExecution, executionUpsert.Previous)
	}
}
