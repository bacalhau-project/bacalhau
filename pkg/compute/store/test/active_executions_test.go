//go:build unit || !integration

package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	ctx            context.Context
	dbFile         string
	executionStore store.ExecutionStore
	execution      store.Execution
}

func (s *Suite) SetupTest() {
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-test")
	s.dbFile = filepath.Join(dir, "test.boltdb")

	s.executionStore, _ = boltdb.NewStore(s.ctx, s.dbFile)
	s.execution = newExecution()
}

func (s *Suite) TearDownTest() {
	os.Remove(s.dbFile)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestGetActiveExecution_Single() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	active, err := store.GetActiveExecution(ctx, s.executionStore, s.execution.Job.ID())
	s.NoError(err)
	s.Equal(s.execution, active)
}

func (s *Suite) TestGetActiveExecution_Multiple() {
	// create a newer execution with same job as the previous one
	newerExecution := s.execution
	newerExecution.ID = uuid.NewString()
	newerExecution.Job = s.execution.Job
	newerExecution.UpdateTime = s.execution.UpdateTime.Add(1)

	err := s.executionStore.CreateExecution(s.ctx, s.execution)
	s.NoError(err)

	err = s.executionStore.CreateExecution(s.ctx, newerExecution)
	s.NoError(err)

	active, err := store.GetActiveExecution(s.ctx, s.executionStore, s.execution.Job.ID())
	s.NoError(err)
	s.Equal(newerExecution, active)
}

func (s *Suite) TestGetActiveExecution_DoestExist() {
	_, err := store.GetActiveExecution(context.Background(), s.executionStore, s.execution.Job.ID())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func newExecution() store.Execution {
	return *store.NewExecution(
		uuid.NewString(),
		model.Job{
			Metadata: model.Metadata{
				ID: uuid.NewString(),
			},
		},
		"nodeID-1",
		model.ResourceUsageData{
			CPU:    1,
			Memory: 2,
		})
}
