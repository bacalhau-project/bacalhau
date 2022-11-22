//go:build unit || !integration

package computenode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	noop_publisher "github.com/filecoin-project/bacalhau/pkg/publisher/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	noop_verifier "github.com/filecoin-project/bacalhau/pkg/verifier/noop"
	"github.com/stretchr/testify/require"
)

func testComputeNode(ctx context.Context, t *testing.T, config ComputeNodeConfig, executorConfig noop_executor.ExecutorConfig) *ComputeNode {
	cm := system.NewCleanupManager()
	defer t.Cleanup(cm.Cleanup)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	var localEventHandler eventhandler.LocalEventHandlerFunc = func(ctx context.Context, event model.JobLocalEvent) error { return nil }
	var eventHandler eventhandler.JobEventHandlerFunc = func(ctx context.Context, event model.JobEvent) error { return nil }

	executor, err := noop_executor.NewNoopExecutorWithConfig(executorConfig)
	require.NoError(t, err)

	verifier, err := noop_verifier.NewNoopVerifier(ctx, cm, localdb.GetStateResolver(datastore))
	require.NoError(t, err)

	publisher, err := noop_publisher.NewNoopPublisher(ctx, cm)
	require.NoError(t, err)

	node, err := NewComputeNode(context.Background(), cm, "test-compute-node",
		datastore,
		localEventHandler,
		eventHandler,
		noop_executor.NewNoopExecutorProvider(executor),
		noop_verifier.NewNoopVerifierProvider(verifier),
		noop_publisher.NewNoopPublisherProvider(publisher),
		config,
	)
	require.NoError(t, err)

	return node
}

func getResources(c, m, d string) model.ResourceUsageConfig {
	return model.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

var normalJob model.Job = model.Job{
	Spec: model.Spec{
		Timeout: time.Minute.Seconds(),
	},
}

// Simple job resource limits tests
func TestJobResourceLimits(t *testing.T) {
	ctx := context.Background()
	runTest := func(t *testing.T, jobResources, jobResourceLimits, defaultJobResourceLimits model.ResourceUsageConfig, expectedResult bool) {
		computeNode := testComputeNode(ctx, t, ComputeNodeConfig{
			CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitJob:            jobResourceLimits,
				ResourceRequirementsDefault: defaultJobResourceLimits,
			},
		}, noop_executor.ExecutorConfig{
			ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
				GetVolumeSize: func(ctx context.Context, volume model.StorageSpec) (uint64, error) {
					return capacitymanager.ConvertMemoryString(jobResources.Disk), nil
				},
			},
		})

		inputs := []model.StorageSpec{}
		if jobResources.Disk != "" {
			inputs = append(inputs, model.StorageSpec{})
		}

		job := model.Job{
			Spec: model.Spec{
				Inputs:    inputs,
				Resources: jobResources,
				Timeout:   time.Second.Seconds(),
			},
		}

		result, _, err := computeNode.SelectJob(ctx, &job)
		require.NoError(t, err)
		require.Equal(t, expectedResult, result, fmt.Sprintf("the expcted result was %v, but got %v -- %+v vs %+v", expectedResult, result, jobResources, jobResourceLimits))
	}

	t.Run("the job is half the limit", func(t *testing.T) {
		runTest(t,
			getResources("1", "500Mb", ""),
			getResources("2", "1Gb", ""),
			getResources("100m", "100Mb", ""),
			true,
		)
	})

	t.Run("the job is on the limit", func(t *testing.T) {
		runTest(t,
			getResources("1", "500Mb", ""),
			getResources("1", "500Mb", ""),
			getResources("100m", "100Mb", ""),
			true,
		)
	})

	t.Run("the job is over the limit", func(t *testing.T) {
		runTest(t,
			getResources("2", "1Gb", ""),
			getResources("1", "500Mb", ""),
			getResources("100m", "100Mb", ""),
			false,
		)
	})

	// test with fractional CPU
	t.Run("the job is less than the limit", func(t *testing.T) {
		runTest(t,
			getResources("250m", "200Mb", ""),
			getResources("1", "500Mb", ""),
			getResources("100m", "100Mb", ""),
			true,
		)
	})

	t.Run("the limit is empty", func(t *testing.T) {
		runTest(t,
			getResources("250m", "200Mb", ""),
			getResources("", "", ""),
			getResources("100m", "100Mb", ""),
			true,
		)
	})

	t.Run("both is empty", func(t *testing.T) {
		runTest(t,
			getResources("", "", ""),
			getResources("", "", ""),
			getResources("100m", "100Mb", ""),
			true,
		)
	})

	t.Run("limit is fractional and under", func(t *testing.T) {
		runTest(t,
			getResources("", "", ""),
			getResources("250m", "200Mb", ""),
			getResources("100m", "100Mb", ""),
			true,
		)
	})

	t.Run("limit is fractional and over", func(t *testing.T) {
		runTest(t,
			getResources("300m", "", ""),
			getResources("250m", "200Mb", ""),
			getResources("100m", "100Mb", ""),
			false,
		)
	})

	t.Run("disk limit is under", func(t *testing.T) {
		runTest(t,
			getResources("", "", "1b"),
			getResources("", "", "6b"),
			getResources("", "", "6b"),
			true,
		)
	})

	t.Run("disk limit is over", func(t *testing.T) {
		runTest(t,
			getResources("", "", "6b"),
			getResources("", "", "1b"),
			getResources("", "", "1b"),
			false,
		)
	})
}

func TestJobSelectionNoVolumes(t *testing.T) {
	ctx := context.Background()
	runTest := func(t *testing.T, rejectSetting, expectedResult bool) {
		stack := testComputeNode(ctx, t, ComputeNodeConfig{
			JobSelectionPolicy: model.JobSelectionPolicy{
				RejectStatelessJobs: rejectSetting,
			},
		}, noop_executor.ExecutorConfig{})

		result, _, err := stack.SelectJob(ctx, &normalJob)
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	t.Run("reject", func(t *testing.T) { runTest(t, true, false) })
	t.Run("accept", func(t *testing.T) { runTest(t, false, true) })
}

// JobSelectionLocality tests that data locality is respected
// when selecting a job
func TestJobSelectionLocality(t *testing.T) {
	ctx := context.Background()
	runTest := func(t *testing.T, locality model.JobSelectionDataLocality, shouldAddData, expectedResult bool) {
		computeNode := testComputeNode(ctx, t,
			ComputeNodeConfig{
				JobSelectionPolicy: model.JobSelectionPolicy{
					Locality: locality,
				},
			},
			noop_executor.ExecutorConfig{
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					HasStorageLocally: func(ctx context.Context, volume model.StorageSpec) (bool, error) { return shouldAddData, nil },
				},
			},
		)

		job := model.Job{
			Spec: model.Spec{
				Engine: model.EngineNoop,
				Inputs: []model.StorageSpec{
					{
						StorageSource: model.StorageSourceIPFS,
						CID:           "abc",
					},
				},
				Timeout: time.Minute.Seconds(),
			},
		}

		result, _, err := computeNode.SelectJob(ctx, &job)
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	// we are local - we do have the file - we should accept
	t.Run("local with file", func(t *testing.T) { runTest(t, model.Local, true, true) })

	// we are local - we don't have the file - we should reject
	t.Run("local without file", func(t *testing.T) { runTest(t, model.Local, false, false) })

	// // we are anywhere - we do have the file - we should accept
	t.Run("anywhere with file", func(t *testing.T) { runTest(t, model.Anywhere, true, true) })

	// // we are anywhere - we don't have the file - we should accept
	t.Run("anywhere without file", func(t *testing.T) { runTest(t, model.Anywhere, false, true) })
}

// TestJobSelectionHttp tests that we can select a job based on
// an http hook
func TestJobSelectionHttp(t *testing.T) {
	ctx := context.Background()
	runTest := func(t *testing.T, failMode, expectedResult bool) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, r.Method, "POST")
			if failMode {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 - Something bad happened!"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("200 - Everything is good!"))
			}

		}))
		defer svr.Close()

		computeNode := testComputeNode(ctx, t, ComputeNodeConfig{
			JobSelectionPolicy: model.JobSelectionPolicy{
				ProbeHTTP: svr.URL,
			},
		}, noop_executor.ExecutorConfig{})

		result, _, err := computeNode.SelectJob(ctx, &normalJob)
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	t.Run("hook says no - we don't accept", func(t *testing.T) { runTest(t, true, false) })
	t.Run("hook says yes - we accept", func(t *testing.T) { runTest(t, false, true) })
}

// TestJobSelectionExec tests that we can select a job based on
// an external command hook
func TestJobSelectionExec(t *testing.T) {
	ctx := context.Background()
	runTest := func(t *testing.T, failMode, expectedResult bool) {
		command := "exit 0"
		if failMode {
			command = "exit 1"
		}
		computeNode := testComputeNode(ctx, t, ComputeNodeConfig{
			JobSelectionPolicy: model.JobSelectionPolicy{
				ProbeExec: command,
			},
		}, noop_executor.ExecutorConfig{})

		result, _, err := computeNode.SelectJob(ctx, &normalJob)
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	t.Run("hook says no - we don't accept", func(t *testing.T) { runTest(t, true, false) })
	t.Run("hook says yes - we accept", func(t *testing.T) { runTest(t, false, true) })
}
