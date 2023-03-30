package testing

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/database"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func truncateTables(t *testing.T, db *gorm.DB, tableNames ...string) {
	for _, table := range tableNames {
		err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error
		if err != nil {
			t.Fatal(err)
		}
	}
}
func newDatabaseStore(t *testing.T) jobstore.Store {
	// empty string creates a new temporary file is created to hold the database.
	// temporary databases are automatically deleted when the connection that created them closes
	dbname := "jobstore.db"
	dial := sqlite.Open(dbname)
	ds, err := database.NewDatabaseStore(dial)
	require.NoError(t, err)
	truncateTables(t, ds.Db, "jobs", "job_states", "job_executions", "execution_outputs", "execution_publish_results", "execution_states", "execution_verification_proposals", "execution_verification_results", "node_executions")
	return ds
}

func newInMemoryStore(t *testing.T) jobstore.Store {
	return inmemory.NewJobStore()
}

func NewStoreBuilder(builders ...builderStore) *StoreBuilder {
	return &StoreBuilder{
		Stores: builders,
	}
}

type builderStore struct {
	Type   string
	Create func(t *testing.T) jobstore.Store
}

type StoreBuilder struct {
	Stores []builderStore
}

func TestJobStore(t *testing.T) {
	ctx := context.Background()
	jobID := "jobid-testing"
	createdAt := time.Unix(0, 0)
	expectedJob := fakeJob(jobID, createdAt)

	sb := NewStoreBuilder(
		builderStore{
			Type:   "InMemory",
			Create: newInMemoryStore,
		},
		builderStore{
			Type:   "Database",
			Create: newDatabaseStore,
		})

	for _, storeBuilder := range sb.Stores {
		t.Run(fmt.Sprintf("%s create job and get job state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			err := store.CreateJob(ctx, expectedJob)
			require.NoError(t, err)

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Empty(t, actualState.Executions)
			// TODO need a mocked clock
			//assert.Equal(t, expectedJob.Metadata.CreatedAt, actualState.CreateTime)
			// assert.Equal(t, actualState.CreateTime.Add(time.Duration(expectedJob.Spec.Timeout)), actualState.TimeoutAt)
			// assert.Equal(t, expectedJob.Metadata.CreatedAt, actualState.UpdateTime)
		})
		t.Run(fmt.Sprintf("%s update job state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			err := store.CreateJob(ctx, expectedJob)
			require.NoError(t, err)

			// initial job state of "new" is updated to "queued"
			initialState := model.JobStateNew
			newState := model.JobStateQueued
			// after an update the expected version should increment
			expectedVersion := 2

			err = store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID: jobID,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState: initialState,
				},
				NewState: newState,
			})
			require.NoError(t, err)

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, newState, actualState.State)
			assert.Equal(t, expectedVersion, actualState.Version)
			assert.Empty(t, actualState.Executions)
			// TODO mock clock
			//assert.Equal(t, expectedJob.Metadata.CreatedAt, actualState.CreateTime)
		})
		t.Run(fmt.Sprintf("%s create execution state and get job state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			expectedJobState := model.JobStateNew
			expectedJobVersion := 1
			err := store.CreateJob(ctx, expectedJob)
			require.NoError(t, err)

			expectedExeNode := "nodeid"
			expectedExeState := model.ExecutionStateNew
			expectedExeVersion := 1
			expectedNumExecutions := 1
			err = store.CreateExecution(ctx, model.ExecutionState{
				JobID:  jobID,
				NodeID: expectedExeNode,
				State:  expectedExeState,
			})
			require.NoError(t, err)

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, expectedJobState, actualState.State)
			assert.Equal(t, expectedJobVersion, actualState.Version)
			assert.Len(t, actualState.Executions, expectedNumExecutions)
			// TODO mock clock
			//assert.Equal(t, expectedJob.Metadata.CreatedAt, actualState.CreateTime)

			actualExecution := actualState.Executions[0]
			assert.Equal(t, jobID, actualExecution.JobID)
			assert.Equal(t, expectedExeState, actualExecution.State)
			assert.Equal(t, expectedExeNode, actualExecution.NodeID)
			assert.Equal(t, expectedExeVersion, actualExecution.Version)
		})
		t.Run(fmt.Sprintf("%s update execution state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			err := store.CreateJob(ctx, expectedJob)
			require.NoError(t, err)

			expectedExecution := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID",
				State:  model.ExecutionStateNew,
			}
			err = store.CreateExecution(ctx, expectedExecution)
			require.NoError(t, err)

			expectedExecutionState := model.ExecutionStateAskForBid
			err = store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
				ExecutionID: expectedExecution.ID(),
				Condition: jobstore.UpdateExecutionCondition{
					ExpectedState:   expectedExecution.State,
					ExpectedVersion: 1,
				},
				NewValues: model.ExecutionState{
					State: expectedExecutionState,
				},
			})
			require.NoError(t, err)

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Len(t, actualState.Executions, 1)
			// TODO mock clock
			//assert.Equal(t, expectedJob.Metadata.CreatedAt, actualState.CreateTime)

			actualExecution := actualState.Executions[0]
			assert.Equal(t, jobID, actualExecution.JobID)
			assert.Equal(t, expectedExecutionState, actualExecution.State)
			assert.Equal(t, expectedExecution.NodeID, actualExecution.NodeID)
			assert.Equal(t, 2, actualExecution.Version)
		})
		t.Run(fmt.Sprintf("%s update execution state with compute reference", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			err := store.CreateJob(ctx, expectedJob)
			require.NoError(t, err)

			expectedExecution := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID",
				State:  model.ExecutionStateNew,
			}
			err = store.CreateExecution(ctx, expectedExecution)
			require.NoError(t, err)

			expectedExecutionState := model.ExecutionStateAskForBid
			err = store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
				ExecutionID: model.ExecutionID{
					JobID:  jobID,
					NodeID: "NodeID",
				},
				Condition: jobstore.UpdateExecutionCondition{
					ExpectedState:   expectedExecution.State,
					ExpectedVersion: 1,
				},
				NewValues: model.ExecutionState{
					State:            expectedExecutionState,
					ComputeReference: "ComputeReference",
				},
			})
			require.NoError(t, err)

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Len(t, actualState.Executions, 1)
			// TODO mock clock
			//assert.Equal(t, expectedJob.Metadata.CreatedAt, actualState.CreateTime)

			actualExecution := actualState.Executions[0]
			assert.Equal(t, jobID, actualExecution.JobID)
			assert.Equal(t, expectedExecutionState, actualExecution.State)
			assert.Equal(t, expectedExecution.NodeID, actualExecution.NodeID)
			assert.Equal(t, 2, actualExecution.Version)
		})
		t.Run(fmt.Sprintf("%s job state with multipule executions", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			err := store.CreateJob(ctx, expectedJob)
			require.NoError(t, err)

			exe1 := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID1",
				State:  model.ExecutionStateNew,
			}
			err = store.CreateExecution(ctx, exe1)
			require.NoError(t, err)

			exe1Update := jobstore.UpdateExecutionRequest{
				ExecutionID: exe1.ID(),
				Condition: jobstore.UpdateExecutionCondition{
					ExpectedState:   exe1.State,
					ExpectedVersion: 1,
				},
				NewValues: model.ExecutionState{
					State: model.ExecutionStateAskForBid,
				},
			}
			err = store.UpdateExecution(ctx, exe1Update)
			require.NoError(t, err)

			exe2 := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID2",
				State:  model.ExecutionStateNew,
			}
			err = store.CreateExecution(ctx, exe2)
			require.NoError(t, err)

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Len(t, actualState.Executions, 2)
			// TODO mock clock
			//assert.Equal(t, expectedJob.Metadata.CreatedAt, actualState.CreateTime)

			// sort by most recent version descending
			sort.Slice(actualState.Executions, func(i, j int) bool {
				return actualState.Executions[i].Version > actualState.Executions[j].Version
			})

			ae1 := actualState.Executions[0]
			ae2 := actualState.Executions[1]
			assert.Equal(t, 2, ae1.Version)
			assert.Equal(t, exe1Update.NewValues.State, ae1.State)

			assert.Equal(t, 1, ae2.Version)
			assert.Equal(t, exe2.State, ae2.State)

		})
	}

}
func fakeJob(id string, createdAt time.Time) model.Job {
	return model.Job{
		APIVersion: "testAPI",
		Metadata: model.Metadata{
			ID:        id,
			CreatedAt: createdAt,
			ClientID:  "ClientID",
			Requester: model.JobRequester{
				RequesterNodeID:    "RequesterNodeID",
				RequesterPublicKey: []byte{1},
			},
		},
	}
}
