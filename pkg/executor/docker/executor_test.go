//go:build unit || !integration

package docker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExecutorTestSuite struct {
	suite.Suite
	executor *Executor
	server   *httptest.Server
	cm       *system.CleanupManager
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}

func (suite *ExecutorTestSuite) SetupTest() {
	docker.MustHaveDocker(suite.T())

	var err error
	suite.cm = system.NewCleanupManager()
	suite.T().Cleanup(suite.cm.Cleanup)

	suite.executor, err = NewExecutor(
		context.Background(),
		suite.cm,
		"bacalhau-executor-unittest",
		storage.NewMappedStorageProvider(map[model.StorageSourceType]storage.Storage{}),
	)
	require.NoError(suite.T(), err)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}

	suite.server = httptest.NewServer(http.HandlerFunc(handler))
	suite.T().Cleanup(suite.server.Close)
}

func (suite *ExecutorTestSuite) containerHttpURL() string {
	url, err := url.Parse(suite.server.URL)
	require.NoError(suite.T(), err)

	// On Mac/Windows, we are within a VM and hence we need to route to the
	// host. On Linux we are not, so localhost should work.
	// See e.g. https://stackoverflow.com/a/24326540
	host := "host.docker.internal"
	if runtime.GOOS == "linux" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%s", host, url.Port())
}

func (suite *ExecutorTestSuite) curlTask() model.JobSpecDocker {
	return model.JobSpecDocker{
		Image:      "curlimages/curl",
		Entrypoint: []string{"curl", path.Join(suite.containerHttpURL(), "hello.txt")},
	}
}

func (suite *ExecutorTestSuite) runJob(spec model.Spec) (*model.RunCommandResult, error) {
	result := suite.T().TempDir()
	j := &model.Job{Metadata: model.Metadata{ID: "test"}, Spec: spec}
	shard := model.JobShard{Job: j, Index: 0}
	return suite.executor.RunShard(context.Background(), shard, result)
}

func (suite *ExecutorTestSuite) runJobGetStdout(spec model.Spec) (string, error) {
	runnerOutput, runErr := suite.runJob(spec)
	return runnerOutput.STDOUT, runErr
}

const (
	CPU_LIMIT    = "100m"
	MEMORY_LIMIT = "100mb"
)

func (suite *ExecutorTestSuite) TestDockerResourceLimitsCPU() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("Resource limits don't apply to containers running on Windows")
	}

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result, err := suite.runJobGetStdout(model.Spec{
		Engine: model.EngineDocker,
		Resources: model.ResourceUsageConfig{
			CPU:    CPU_LIMIT,
			Memory: MEMORY_LIMIT,
		},
		Docker: model.JobSpecDocker{
			Image:      "ubuntu",
			Entrypoint: []string{"bash", "-c", "cat /sys/fs/cgroup/cpu.max"},
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
	if runtime.GOOS == "windows" {
		suite.T().Skip("Resource limits don't apply to containers running on Windows")
	}

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result, err := suite.runJobGetStdout(model.Spec{
		Engine: model.EngineDocker,
		Resources: model.ResourceUsageConfig{
			CPU:    CPU_LIMIT,
			Memory: MEMORY_LIMIT,
		},
		Docker: model.JobSpecDocker{
			Image:      "ubuntu",
			Entrypoint: []string{"bash", "-c", "cat /sys/fs/cgroup/memory.max"},
		},
	})
	require.NoError(suite.T(), err)

	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), capacity.ConvertBytesString(MEMORY_LIMIT), uint64(intVar), "the container reported memory does not equal the configured limit")
}

func (suite *ExecutorTestSuite) TestDockerNetworkingFull() {
	result, err := suite.runJob(model.Spec{
		Engine:  model.EngineDocker,
		Network: model.NetworkConfig{Type: model.NetworkFull},
		Docker:  suite.curlTask(),
	})
	require.NoError(suite.T(), err, result.STDERR)
	require.Equal(suite.T(), "/hello.txt", result.STDOUT)
}

func (suite *ExecutorTestSuite) TestDockerNetworkingNone() {
	result, err := suite.runJob(model.Spec{
		Engine:  model.EngineDocker,
		Network: model.NetworkConfig{Type: model.NetworkNone},
		Docker:  suite.curlTask(),
	})
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), result.STDOUT)
	require.NotEmpty(suite.T(), result.STDERR)
	require.NotZero(suite.T(), result.ExitCode)
}
