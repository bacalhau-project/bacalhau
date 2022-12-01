//go:build unit || (!integration && !windows)

package docker

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExecutorTestSuite struct {
	suite.Suite
	executor *Executor
	cm       *system.CleanupManager
}

func (suite *ExecutorTestSuite) SetupSuite() {
	var err error
	suite.cm = system.NewCleanupManager()
	suite.executor, err = NewExecutor(
		context.Background(),
		suite.cm,
		"bacalhau-executor-unittest",
		storage.NewMappedStorageProvider(map[model.StorageSourceType]storage.Storage{}),
	)
	require.NoError(suite.T(), err)
}

func (suite *ExecutorTestSuite) TearDownTest() {
	suite.cm.Cleanup()
}

func (suite *ExecutorTestSuite) TestDockerResourceLimitsCPU() {
	ctx := context.Background()
	CPU_LIMIT := "100m"

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result := runJobGetStdout(ctx, suite.T(), suite.executor, model.Spec{
		Engine:   model.EngineDocker,
		Verifier: model.VerifierNoop,
		Resources: model.ResourceUsageConfig{
			CPU:    CPU_LIMIT,
			Memory: "100mb",
		},
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"cat /sys/fs/cgroup/cpu.max",
			},
		},
	})

	values := strings.Fields(result)

	numerator, err := strconv.Atoi(values[0])
	require.NoError(suite.T(), err)

	denominator, err := strconv.Atoi(values[1])
	require.NoError(suite.T(), err)

	var containerCPU float64 = 0

	if denominator > 0 {
		containerCPU = float64(numerator) / float64(denominator)
	}

	require.Equal(suite.T(), capacity.ConvertCPUString(CPU_LIMIT), containerCPU, "the container reported CPU does not equal the configured limit")
}

func (suite *ExecutorTestSuite) TestDockerResourceLimitsMemory() {
	ctx := context.Background()
	MEMORY_LIMIT := "100mb"

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result := runJobGetStdout(ctx, suite.T(), suite.executor, model.Spec{
		Engine:   model.EngineDocker,
		Verifier: model.VerifierNoop,
		Resources: model.ResourceUsageConfig{
			CPU:    "100m",
			Memory: MEMORY_LIMIT,
		},
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"cat /sys/fs/cgroup/memory.max",
			},
		},
	})

	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), capacity.ConvertBytesString(MEMORY_LIMIT), uint64(intVar), "the container reported memory does not equal the configured limit")
}

func runJobGetStdout(
	ctx context.Context,
	t *testing.T,
	executor *Executor,
	spec model.Spec,
) string {
	result := t.TempDir()

	j := &model.Job{
		ID:   "test",
		Spec: spec,
	}
	shard := model.JobShard{
		Job:   j,
		Index: 0,
	}

	runnerOutput, err := executor.RunShard(ctx, shard, result)
	require.NoError(t, err)
	require.Empty(t, runnerOutput.ErrorMsg)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.DirExists(t, result, "The job result folder exists")
	require.FileExists(t, stdoutPath, "The stdout file exists")
	dat, err := os.ReadFile(stdoutPath)
	require.NoError(t, err)
	return string(dat)
}
