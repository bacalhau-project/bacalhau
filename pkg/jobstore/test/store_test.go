//go:build unit || !integration

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/raulk/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore/persistent"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func newDatabaseStore(t *testing.T) jobstore.Store {
	ds, err := persistent.NewStore()
	require.NoError(t, err)
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

func TestJobStoreErrorCases(t *testing.T) {

	ctx := context.Background()

	sb := NewStoreBuilder(
		builderStore{
			Type:   "InMemory",
			Create: newInMemoryStore,
		},
		builderStore{
			Type:   "Persistent",
			Create: newDatabaseStore,
		})

	mockClock := clock.NewMock()
	jobID := "jobid-testing"
	expectedJob := fakeJob(jobID, mockClock.Now())

	for _, storeBuilder := range sb.Stores {
		t.Run(fmt.Sprintf("%s return error for duplicate jobIDs", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)
			assert.NoError(t, store.CreateJob(ctx, expectedJob))
			assert.ErrorIs(t, jobstore.NewErrJobAlreadyExists(expectedJob.ID()), store.CreateJob(ctx, expectedJob))
		})

		t.Run(fmt.Sprintf("%s return error for job not found on get", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)
			noJobID, actualErr := store.GetJob(ctx, expectedJob.ID())
			assert.Equal(t, "", noJobID.ID())
			// TODO in memory store is inconsistent with error types returned.
			if storeBuilder.Type == "InMemory" {
				assert.Error(t, actualErr)
			} else {
				assert.ErrorIs(t, actualErr, jobstore.NewErrJobNotFound(expectedJob.ID()))
			}
		})

		t.Run(fmt.Sprintf("%s return error for job not found on update", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)
			actualErr := store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID:    expectedJob.ID(),
				NewState: model.JobStateQueued,
				Comment:  "",
			})
			// TODO in memory store is inconsistent with error types returned.
			if storeBuilder.Type == "InMemory" {
				assert.Error(t, actualErr)
			} else {
				assert.ErrorIs(t, actualErr, jobstore.NewErrJobNotFound(expectedJob.ID()))
			}
		})

		t.Run(fmt.Sprintf("%s return error for job not found when creating execution with no job", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)
			assert.Error(t, store.CreateExecution(ctx, "executionID", model.ExecutionState{
				JobID:            expectedJob.ID(),
				NodeID:           "NodeID",
				ComputeReference: "executionID",
			}))
		})

		t.Run(fmt.Sprintf("%s return error for job not found when updating execution with no job", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)
			assert.Error(t, store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
				ExecutionID: model.ExecutionID{
					JobID:       expectedJob.ID(),
					NodeID:      "NodeID",
					ExecutionID: "ExecutionID",
				},
				Condition: jobstore.UpdateExecutionCondition{},
				NewValues: model.ExecutionState{},
				Comment:   "",
			}))
		})
	}
}

func TestJobStoreHappyPath(t *testing.T) {
	ctx := context.Background()

	sb := NewStoreBuilder(
		builderStore{
			Type:   "InMemory",
			Create: newInMemoryStore,
		},
		builderStore{
			Type:   "Persistent",
			Create: newDatabaseStore,
		})

	mockClock := clock.NewMock()
	jobID := "jobid-testing"
	expectedJob := fakeJob(jobID, mockClock.Now())

	for _, storeBuilder := range sb.Stores {
		t.Run(fmt.Sprintf("%s create job and get job state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			require.NoError(t, store.CreateJob(ctx, expectedJob))

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, expectedJob.ID(), actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Empty(t, actualState.Executions)
		})

		t.Run(fmt.Sprintf("%s update job state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			require.NoError(t, store.CreateJob(ctx, expectedJob))

			require.NoError(t, store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID: jobID,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState: model.JobStateNew,
				},
				NewState: model.JobStateQueued,
			}))

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateQueued, actualState.State)
			assert.Equal(t, 2, actualState.Version)
			assert.Empty(t, actualState.Executions)
		})

		t.Run(fmt.Sprintf("%s create execution state and get job state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			require.NoError(t, store.CreateJob(ctx, expectedJob))

			executionID1 := "exe-id-1"
			require.NoError(t, store.CreateExecution(ctx, executionID1, model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID",
				State:  model.ExecutionStateNew,
			}))

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Len(t, actualState.Executions, 1)

			actualExecution := actualState.Executions[0]
			assert.Equal(t, jobID, actualExecution.JobID)
			assert.Equal(t, model.ExecutionStateNew, actualExecution.State)
			assert.Equal(t, "NodeID", actualExecution.NodeID)
			assert.Equal(t, 1, actualExecution.Version)
		})

		t.Run(fmt.Sprintf("%s update execution state", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			require.NoError(t, store.CreateJob(ctx, expectedJob))

			expectedExecution := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID",
				State:  model.ExecutionStateNew,
			}
			executionID1 := "exe-id-1"
			require.NoError(t, store.CreateExecution(ctx, executionID1, expectedExecution))

			expectedExecutionState := model.ExecutionStateAskForBid
			require.NoError(t, store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
				ExecutionID: model.ExecutionID{
					JobID:       jobID,
					NodeID:      "NodeID",
					ExecutionID: executionID1,
				},
				Condition: jobstore.UpdateExecutionCondition{
					ExpectedState:   expectedExecution.State,
					ExpectedVersion: 1,
				},
				NewValues: model.ExecutionState{
					State: expectedExecutionState,
				},
			}))

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Len(t, actualState.Executions, 1)

			actualExecution := actualState.Executions[0]
			assert.Equal(t, jobID, actualExecution.JobID)
			assert.Equal(t, expectedExecutionState, actualExecution.State)
			assert.Equal(t, expectedExecution.NodeID, actualExecution.NodeID)
			assert.Equal(t, 2, actualExecution.Version)
		})

		t.Run(fmt.Sprintf("%s job state with multipule executions", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			require.NoError(t, store.CreateJob(ctx, expectedJob))

			exe1 := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID1",
				State:  model.ExecutionStateNew,
			}
			executionID1 := "exe-id-1"
			require.NoError(t, store.CreateExecution(ctx, executionID1, exe1))

			exe1Update := jobstore.UpdateExecutionRequest{
				ExecutionID: model.ExecutionID{
					JobID:       jobID,
					NodeID:      "NodeID1",
					ExecutionID: executionID1,
				},
				Condition: jobstore.UpdateExecutionCondition{
					ExpectedState:   exe1.State,
					ExpectedVersion: 1,
				},
				NewValues: model.ExecutionState{
					State: model.ExecutionStateAskForBid,
				},
			}
			require.NoError(t, store.UpdateExecution(ctx, exe1Update))

			exe2 := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID2",
				State:  model.ExecutionStateNew,
			}
			executionID2 := "exe-id-2"
			require.NoError(t, store.CreateExecution(ctx, executionID2, exe2))

			actualState, err := store.GetJobState(ctx, jobID)
			require.NoError(t, err)

			assert.Equal(t, jobID, actualState.JobID)
			assert.Equal(t, model.JobStateNew, actualState.State)
			assert.Equal(t, 1, actualState.Version)
			assert.Len(t, actualState.Executions, 2)

			ae1 := actualState.Executions[0]
			ae2 := actualState.Executions[1]
			assert.Equal(t, 2, ae1.Version)
			assert.Equal(t, exe1Update.NewValues.State, ae1.State)

			assert.Equal(t, 1, ae2.Version)
			assert.Equal(t, exe2.State, ae2.State)
		})

		t.Run(fmt.Sprintf("%s get in progress jobs", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			jobID1 := "jobid-testing-1"
			job1 := fakeJob(jobID1, mockClock.Now())
			require.NoError(t, store.CreateJob(ctx, job1))

			jobID2 := "jobid-testing-2"
			job2 := fakeJob(jobID2, job1.Metadata.CreatedAt.Add(time.Second))
			require.NoError(t, store.CreateJob(ctx, job2))
			require.NoError(t, store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID: jobID2,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState:   model.JobStateNew,
					ExpectedVersion: 1,
				},
				NewState: model.JobStateCancelled,
				Comment:  "",
			}))

			jobID3 := "jobid-testing-3"
			job3 := fakeJob(jobID3, job2.Metadata.CreatedAt.Add(time.Second))
			require.NoError(t, store.CreateJob(ctx, job3))
			require.NoError(t, store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID: jobID3,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState:   model.JobStateNew,
					ExpectedVersion: 1,
				},
				NewState: model.JobStateInProgress,
				Comment:  "",
			}))

			jobID4 := "jobid-testing-4"
			job4 := fakeJob(jobID4, job3.Metadata.CreatedAt.Add(time.Second))
			require.NoError(t, store.CreateJob(ctx, job4))
			require.NoError(t, store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID: jobID4,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState:   model.JobStateNew,
					ExpectedVersion: 1,
				},
				NewState: model.JobStateQueued,
				Comment:  "",
			}))

			jobID5 := "jobid-testing-5"
			job5 := fakeJob(jobID5, job4.Metadata.CreatedAt.Add(time.Second))
			require.NoError(t, store.CreateJob(ctx, job5))
			require.NoError(t, store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID: jobID5,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState:   model.JobStateNew,
					ExpectedVersion: 1,
				},
				NewState: model.JobStateCompleted,
				Comment:  "",
			}))

			inProgressJobs, err := store.GetInProgressJobs(ctx)
			require.NoError(t, err)
			assert.Len(t, inProgressJobs, 3)
			assert.Equal(t, jobID1, inProgressJobs[0].Job.ID())
			assert.Equal(t, jobID3, inProgressJobs[1].Job.ID())
			assert.Equal(t, jobID4, inProgressJobs[2].Job.ID())

		})

		t.Run(fmt.Sprintf("%s get job history", storeBuilder.Type), func(t *testing.T) {
			store := storeBuilder.Create(t)

			historyClock := clock.NewMock()
			historyClock.Set(expectedJob.Metadata.CreatedAt)
			createJobTime := historyClock.Now()
			require.NoError(t, store.CreateJob(ctx, expectedJob))

			createExe1Time := createJobTime.Add(time.Second)
			historyClock.Set(createExe1Time)
			exe1 := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID1",
				State:  model.ExecutionStateNew,
			}
			executionID1 := "exe-id-1"
			require.NoError(t, store.CreateExecution(ctx, executionID1, exe1))

			createExe2Time := createExe1Time.Add(time.Second)
			historyClock.Set(createExe2Time)
			exe2 := model.ExecutionState{
				JobID:  jobID,
				NodeID: "NodeID2",
				State:  model.ExecutionStateNew,
			}
			executionID2 := "exe-id0"
			require.NoError(t, store.CreateExecution(ctx, executionID2, exe2))

			updateExe2Time := createExe2Time.Add(time.Second)
			historyClock.Set(updateExe2Time)
			require.NoError(t, store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
				ExecutionID: model.ExecutionID{
					JobID:       jobID,
					NodeID:      "NodeID2",
					ExecutionID: executionID2,
				},
				NewValues: model.ExecutionState{
					State: model.ExecutionStateAskForBid,
				},
				Comment: "asking for bid",
			}))

			updateJobStateTime := updateExe2Time.Add(time.Second)
			historyClock.Set(updateJobStateTime)
			require.NoError(t, store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
				JobID:    jobID,
				NewState: model.JobStateInProgress,
				Comment:  "job in progress",
			}))

			history, err := store.GetJobHistory(ctx, jobID, jobstore.JobHistoryFilterOptions{
				Since:                 0,
				ExcludeExecutionLevel: false,
				ExcludeJobLevel:       false,
			})
			require.NoError(t, err)
			// create job
			// create execution1
			// create execution2
			// update execution1
			// update job
			assert.Len(t, history, 5)

			assert.Equal(t, &model.StateChange[model.JobStateType]{
				Previous: model.JobStateNew,
				New:      model.JobStateNew,
			}, history[0].JobState)

			assert.Equal(t, &model.StateChange[model.ExecutionStateType]{
				Previous: model.ExecutionStateNew,
				New:      model.ExecutionStateNew,
			}, history[1].ExecutionState)

			assert.Equal(t, &model.StateChange[model.ExecutionStateType]{
				Previous: model.ExecutionStateNew,
				New:      model.ExecutionStateNew,
			}, history[2].ExecutionState)

			assert.Equal(t, &model.StateChange[model.ExecutionStateType]{
				Previous: model.ExecutionStateNew,
				New:      model.ExecutionStateAskForBid,
			}, history[3].ExecutionState)

			assert.Equal(t, &model.StateChange[model.JobStateType]{
				Previous: model.JobStateNew,
				New:      model.JobStateInProgress,
			}, history[4].JobState)

			// reuse state from above test to extend test case.
			t.Run(fmt.Sprintf("%s get job history with multipule jobs", storeBuilder.Type), func(t *testing.T) {
				createJobTime := historyClock.Now()
				jobID2 := "jobid-testin-2"
				expectedJob2 := fakeJob(jobID2, mockClock.Now())
				require.NoError(t, store.CreateJob(ctx, expectedJob2))

				createExe1Time := createJobTime.Add(time.Second)
				historyClock.Set(createExe1Time)
				exe1 := model.ExecutionState{
					JobID:  jobID2,
					NodeID: "NodeID1",
					State:  model.ExecutionStateNew,
				}
				executionID1 := "exe-id-1"
				require.NoError(t, store.CreateExecution(ctx, executionID1, exe1))

				createExe2Time := createExe1Time.Add(time.Second)
				historyClock.Set(createExe2Time)
				exe2 := model.ExecutionState{
					JobID:  jobID2,
					NodeID: "NodeID2",
					State:  model.ExecutionStateNew,
				}
				executionID2 := "exe-id0"
				require.NoError(t, store.CreateExecution(ctx, executionID2, exe2))

				updateExe2Time := createExe2Time.Add(time.Second)
				historyClock.Set(updateExe2Time)
				require.NoError(t, store.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
					ExecutionID: model.ExecutionID{
						JobID:       jobID2,
						NodeID:      "NodeID2",
						ExecutionID: executionID2,
					},
					NewValues: model.ExecutionState{
						State: model.ExecutionStateAskForBid,
					},
					Comment: "asking for bid",
				}))

				updateJobStateTime := updateExe2Time.Add(time.Second)
				historyClock.Set(updateJobStateTime)
				require.NoError(t, store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
					JobID:    jobID2,
					NewState: model.JobStateInProgress,
					Comment:  "job in progress",
				}))

				history, err := store.GetJobHistory(ctx, jobID2, jobstore.JobHistoryFilterOptions{
					Since:                 0,
					ExcludeExecutionLevel: false,
					ExcludeJobLevel:       false,
				})
				require.NoError(t, err)
				// create job
				// create execution1
				// create execution2
				// update execution1
				// update job
				assert.Len(t, history, 5)

				assert.Equal(t, &model.StateChange[model.JobStateType]{
					Previous: model.JobStateNew,
					New:      model.JobStateNew,
				}, history[0].JobState)

				assert.Equal(t, &model.StateChange[model.ExecutionStateType]{
					Previous: model.ExecutionStateNew,
					New:      model.ExecutionStateNew,
				}, history[1].ExecutionState)

				assert.Equal(t, &model.StateChange[model.ExecutionStateType]{
					Previous: model.ExecutionStateNew,
					New:      model.ExecutionStateNew,
				}, history[2].ExecutionState)

				assert.Equal(t, &model.StateChange[model.ExecutionStateType]{
					Previous: model.ExecutionStateNew,
					New:      model.ExecutionStateAskForBid,
				}, history[3].ExecutionState)

				assert.Equal(t, &model.StateChange[model.JobStateType]{
					Previous: model.JobStateNew,
					New:      model.JobStateInProgress,
				}, history[4].JobState)

				t.Run(fmt.Sprintf("%s get job history with multipule jobs excluding execution state", storeBuilder.Type), func(t *testing.T) {
					history, err := store.GetJobHistory(ctx, jobID2, jobstore.JobHistoryFilterOptions{
						Since:                 0,
						ExcludeExecutionLevel: true,
						ExcludeJobLevel:       false,
					})
					require.NoError(t, err)
					// create job
					// create execution1 excluded
					// create execution2 excluded
					// update execution1 excluded
					// update job
					assert.Len(t, history, 2)

					assert.Equal(t, &model.StateChange[model.JobStateType]{
						Previous: model.JobStateNew,
						New:      model.JobStateNew,
					}, history[0].JobState)

					assert.Equal(t, &model.StateChange[model.JobStateType]{
						Previous: model.JobStateNew,
						New:      model.JobStateInProgress,
					}, history[1].JobState)
				})

			})
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
