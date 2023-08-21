//go:build unit || !integration

package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	ctx                 context.Context
	dbFile              string
	executionStore      store.ExecutionStore
	localExecutionState store.LocalExecutionState
	execution           *models.Execution
}

func (s *Suite) SetupTest() {
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-test")
	s.dbFile = filepath.Join(dir, "test.boltdb")

	s.executionStore, _ = boltdb.NewStore(s.ctx, s.dbFile)
	s.execution = mock.ExecutionForJob(mock.Job())
	s.localExecutionState = *store.NewLocalExecutionState(s.execution, "nodeID-1")
}

func (s *Suite) TearDownTest() {
	os.Remove(s.dbFile)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestGetActiveExecution_Single() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.NoError(err)

	active, err := store.GetActiveExecution(ctx, s.executionStore, s.execution.JobID)
	s.NoError(err)
	s.Equal(s.localExecutionState, active)
}

func (s *Suite) TestGetActiveExecution_Multiple() {
	// create a newer execution with same job as the previous one
	newerExecution := s.execution.Copy()
	newerExecution.ID = uuid.NewString()
	newerExecution.ModifyTime = s.execution.ModifyTime + 1

	newerExecutionState := *store.NewLocalExecutionState(newerExecution, "nodeID-1")

	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.NoError(err)

	err = s.executionStore.CreateExecution(s.ctx, newerExecutionState)
	s.NoError(err)

	active, err := store.GetActiveExecution(s.ctx, s.executionStore, s.execution.JobID)
	s.NoError(err)
	s.Equal(newerExecutionState, active)
}

func (s *Suite) TestGetActiveExecution_DoestExist() {
	_, err := store.GetActiveExecution(context.Background(), s.executionStore, s.execution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}
