//go:build unit || !integration

package docker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

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
	server   *http.Server
	cm       *system.CleanupManager
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}

func (s *ExecutorTestSuite) SetupTest() {
	docker.MustHaveDocker(s.T())

	var err error
	s.cm = system.NewCleanupManager()
	s.T().Cleanup(func() {
		s.cm.Cleanup(context.Background())
	})

	s.executor, err = NewExecutor(
		context.Background(),
		s.cm,
		"bacalhau-executor-unittest",
		model.NewMappedProvider(map[model.StorageSourceType]storage.Storage{}),
	)
	require.NoError(s.T(), err)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}

	// We have to manually discover the correct IP address for the server to
	// listen on because on Linux hosts simply using 127.0.0.1 will get caught
	// in the loopback interface of the gateway container. We have to listen on
	// whatever "host.docker.internal" resolves to, which is the IP address of
	// the "docker0" interface.
	var gateway net.IP
	if runtime.GOOS == "linux" {
		gateway, err = s.executor.client.HostGatewayIP(context.Background())
		require.NoError(s.T(), err)
	} else {
		gateway = net.ParseIP("127.0.0.1")
	}

	serverAddr := net.TCPAddr{IP: gateway, Port: 0}
	listener, err := net.Listen("tcp", serverAddr.String())
	require.NoError(s.T(), err)
	// Don't need to close the listener as it'll be closed by the server.

	s.server = &http.Server{
		Addr:    listener.Addr().String(),
		Handler: http.HandlerFunc(handler),
	}
	s.cm.RegisterCallback(s.server.Close)
	go s.server.Serve(listener)
}

func (s *ExecutorTestSuite) containerHttpURL() *url.URL {
	url, err := url.Parse("http://" + s.server.Addr)
	require.NoError(s.T(), err)

	// On Mac/Windows, we are within a VM and hence we need to route to the
	// host. On Linux we are not, so localhost should work.
	// See e.g. https://stackoverflow.com/a/24326540
	url.Host = fmt.Sprintf("%s:%s", dockerHostHostname, url.Port())
	return url
}

func (s *ExecutorTestSuite) curlTask() model.JobSpecDocker {
	return model.JobSpecDocker{
		Image:      "curlimages/curl",
		Entrypoint: []string{"curl", "--fail-with-body", s.containerHttpURL().JoinPath("hello.txt").String()},
	}
}

func (s *ExecutorTestSuite) runJob(spec model.Spec) (*model.RunCommandResult, error) {
	return s.runJobWithContext(context.Background(), spec)
}

func (s *ExecutorTestSuite) runJobWithContext(ctx context.Context, spec model.Spec) (*model.RunCommandResult, error) {
	result := s.T().TempDir()
	j := &model.Job{Metadata: model.Metadata{ID: "test"}, Spec: spec}
	shard := model.JobShard{Job: j, Index: 0}
	return s.executor.RunShard(ctx, shard, result)
}

func (s *ExecutorTestSuite) runJobGetStdout(spec model.Spec) (string, error) {
	runnerOutput, runErr := s.runJob(spec)
	return runnerOutput.STDOUT, runErr
}

const (
	CPU_LIMIT    = "100m"
	MEMORY_LIMIT = "100mb"
)

func (s *ExecutorTestSuite) TestDockerResourceLimitsCPU() {
	if runtime.GOOS == "windows" {
		s.T().Skip("Resource limits don't apply to containers running on Windows")
	}

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result, err := s.runJobGetStdout(model.Spec{
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
	require.NoError(s.T(), err)

	denominator, err := strconv.Atoi(values[1])
	require.NoError(s.T(), err)

	var containerCPU float64 = 0

	if denominator > 0 {
		containerCPU = float64(numerator) / float64(denominator)
	}

	require.Equal(s.T(), capacity.ConvertCPUString(CPU_LIMIT), containerCPU, "the container reported CPU does not equal the configured limit")
}

func (s *ExecutorTestSuite) TestDockerResourceLimitsMemory() {
	if runtime.GOOS == "windows" {
		s.T().Skip("Resource limits don't apply to containers running on Windows")
	}

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	result, err := s.runJobGetStdout(model.Spec{
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
	require.NoError(s.T(), err)

	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(s.T(), err)
	require.Equal(s.T(), capacity.ConvertBytesString(MEMORY_LIMIT), uint64(intVar), "the container reported memory does not equal the configured limit")
}

func (s *ExecutorTestSuite) TestDockerNetworkingFull() {
	result, err := s.runJob(model.Spec{
		Engine:  model.EngineDocker,
		Network: model.NetworkConfig{Type: model.NetworkFull},
		Docker:  s.curlTask(),
	})
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingNone() {
	result, err := s.runJob(model.Spec{
		Engine:  model.EngineDocker,
		Network: model.NetworkConfig{Type: model.NetworkNone},
		Docker:  s.curlTask(),
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), result.STDOUT)
	require.NotEmpty(s.T(), result.STDERR)
	require.NotZero(s.T(), result.ExitCode)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTP() {
	result, err := s.runJob(model.Spec{
		Engine: model.EngineDocker,
		Network: model.NetworkConfig{
			Type:    model.NetworkHTTP,
			Domains: []string{s.containerHttpURL().Hostname()},
		},
		Docker: s.curlTask(),
	})
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTPWithMultipleDomains() {
	result, err := s.runJob(model.Spec{
		Engine: model.EngineDocker,
		Network: model.NetworkConfig{
			Type: model.NetworkHTTP,
			Domains: []string{
				s.containerHttpURL().Hostname(),
				"bacalhau.org",
			},
		},
		Docker: s.curlTask(),
	})
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingWithSubdomains() {
	hostname := s.containerHttpURL().Hostname()
	hostroot := strings.Join(strings.SplitN(hostname, ".", 2)[:1], ".")

	result, err := s.runJob(model.Spec{
		Engine: model.EngineDocker,
		Network: model.NetworkConfig{
			Type:    model.NetworkHTTP,
			Domains: []string{hostname, hostroot},
		},
		Docker: s.curlTask(),
	})
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingFiltersHTTP() {
	result, err := s.runJob(model.Spec{
		Engine: model.EngineDocker,
		Network: model.NetworkConfig{
			Type:    model.NetworkHTTP,
			Domains: []string{"bacalhau.org"},
		},
		Docker: s.curlTask(),
	})
	// The curl will succeed but should return a non-zero exit code and error page.
	require.NoError(s.T(), err)
	require.NotZero(s.T(), result.ExitCode)
	require.Contains(s.T(), result.STDOUT, "ERROR: The requested URL could not be retrieved")
}

func (s *ExecutorTestSuite) TestDockerNetworkingFiltersHTTPS() {
	result, err := s.runJob(model.Spec{
		Engine: model.EngineDocker,
		Network: model.NetworkConfig{
			Type:    model.NetworkHTTP,
			Domains: []string{s.containerHttpURL().Hostname()},
		},
		Docker: model.JobSpecDocker{
			Image:      "curlimages/curl",
			Entrypoint: []string{"curl", "--fail-with-body", "https://www.bacalhau.org"},
		},
	})
	// The curl will succeed but should return a non-zero exit code and error page.
	require.NoError(s.T(), err)
	require.NotZero(s.T(), result.ExitCode)
	require.Empty(s.T(), result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingAppendsHTTPHeader() {
	s.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(r.Header.Get("X-Bacalhau-Job-ID")))
		s.Require().NoError(err)
	})
	result, err := s.runJob(model.Spec{
		Engine: model.EngineDocker,
		Network: model.NetworkConfig{
			Type:    model.NetworkHTTP,
			Domains: []string{s.containerHttpURL().Hostname()},
		},
		Docker: s.curlTask(),
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), "test", result.STDOUT, result.STDOUT)
}

func (s *ExecutorTestSuite) TestTimesOutCorrectly() {
	expected := "message after sleep"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := s.runJobWithContext(ctx, model.Spec{
		Engine: model.EngineDocker,
		Docker: model.JobSpecDocker{
			Image:      "ubuntu",
			Entrypoint: []string{"bash", "-c", fmt.Sprintf(`sleep 1 && echo "%s" && sleep 20`, expected)},
		},
	})
	s.ErrorIs(err, context.DeadlineExceeded)
	s.Truef(strings.HasPrefix(result.STDOUT, expected), "'%s' does not start with '%s'", result.STDOUT, expected)
}
