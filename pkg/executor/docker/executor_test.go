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

	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const (
	CurlDockerImageName string = "curlimages/curl"
	CurlDockerImageTag  string = "8.1.0"
	CurlDockerImage     string = CurlDockerImageName + ":" + CurlDockerImageTag
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

func (s *ExecutorTestSuite) curlTask() *models.SpecConfig {
	return dockermodels.NewDockerEngineBuilder(CurlDockerImage).
		WithEntrypoint("curl", "--fail-with-body", s.containerHttpURL().JoinPath("hello.txt").String()).
		Build()
}

func (s *ExecutorTestSuite) runJob(spec *models.Task) (*models.RunCommandResult, error) {
	return s.runJobWithContext(context.Background(), spec, "test")
}

func (s *ExecutorTestSuite) runJobWithContext(ctx context.Context, spec *models.Task, name string) (*models.RunCommandResult, error) {
	result := s.T().TempDir()
	j := mock.Job()
	j.ID = name
	j.Tasks = []*models.Task{spec}

	resources, err := spec.ResourcesConfig.ToResources()
	require.NoError(s.T(), err)

	return s.executor.Run(
		ctx,
		&executor.RunCommandRequest{
			JobID:        j.ID,
			ExecutionID:  name,
			Resources:    resources,
			Network:      spec.Network,
			Outputs:      spec.ResultPaths,
			Inputs:       nil,
			ResultsDir:   result,
			EngineParams: spec.Engine,
			OutputLimits: executor.OutputLimits{
				MaxStdoutFileLength:   system.MaxStdoutFileLength,
				MaxStdoutReturnLength: system.MaxStdoutReturnLength,
				MaxStderrFileLength:   system.MaxStderrFileLength,
				MaxStderrReturnLength: system.MaxStderrReturnLength,
			},
		},
	)
}

func (s *ExecutorTestSuite) runJobGetStdout(spec *models.Task) (string, error) {
	runnerOutput, runErr := s.runJob(spec)
	return runnerOutput.STDOUT, runErr
}

const (
	CPU_LIMIT    = "100m"
	MEMORY_LIMIT = "100mb"

	CPU_LIMIT_UNITS    = 0.1
	MEMORY_LIMIT_BYTES = 100 * 1024 * 1024
)

func (s *ExecutorTestSuite) TestDockerResourceLimitsCPU() {
	if runtime.GOOS == "windows" {
		s.T().Skip("Resource limits don't apply to containers running on Windows")
	}

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications

	task := mock.TaskBuilder().
		Engine(dockermodels.NewDockerEngineBuilder("ubuntu").
			WithEntrypoint("bash", "-c", "cat /sys/fs/cgroup/cpu.max").
			Build()).
		ResourcesConfig(models.NewResourcesConfigBuilder().CPU(CPU_LIMIT).Memory(MEMORY_LIMIT).BuildOrDie()).
		BuildOrDie()

	result, err := s.runJobGetStdout(task)
	require.NoError(s.T(), err)

	values := strings.Fields(result)

	numerator, err := strconv.Atoi(values[0])
	require.NoError(s.T(), err)

	denominator, err := strconv.Atoi(values[1])
	require.NoError(s.T(), err)

	var containerCPU float64 = 0

	if denominator > 0 {
		containerCPU = float64(numerator) / float64(denominator)
	}

	require.Equal(s.T(), CPU_LIMIT_UNITS, containerCPU, "the container reported CPU does not equal the configured limit")
}

func (s *ExecutorTestSuite) TestDockerResourceLimitsMemory() {
	if runtime.GOOS == "windows" {
		s.T().Skip("Resource limits don't apply to containers running on Windows")
	}

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications

	task := mock.TaskBuilder().
		Engine(
			dockermodels.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint("bash", "-c", "cat /sys/fs/cgroup/memory.max").
				Build()).
		ResourcesConfig(models.NewResourcesConfigBuilder().CPU(CPU_LIMIT).Memory(MEMORY_LIMIT).BuildOrDie()).
		BuildOrDie()

	result, err := s.runJobGetStdout(task)
	require.NoError(s.T(), err)

	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(s.T(), err)
	require.Equal(s.T(), MEMORY_LIMIT_BYTES, intVar, "the container reported memory does not equal the configured limit")
}

func (s *ExecutorTestSuite) TestDockerNetworkingFull() {
	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().
			Type(models.NetworkFull).
			BuildOrDie()).
		Engine(s.curlTask()).
		BuildOrDie()

	result, err := s.runJob(task)
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingNone() {
	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().
			Type(models.NetworkNone).
			BuildOrDie()).
		Engine(s.curlTask()).
		BuildOrDie()

	result, err := s.runJob(task)
	require.NoError(s.T(), err)
	require.Empty(s.T(), result.STDOUT)
	require.NotEmpty(s.T(), result.STDERR)
	require.NotZero(s.T(), result.ExitCode)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTP() {
	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().
			Type(models.NetworkHTTP).
			Domains(s.containerHttpURL().Hostname()).
			BuildOrDie()).
		Engine(s.curlTask()).
		BuildOrDie()

	result, err := s.runJob(task)
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTPWithMultipleDomains() {
	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().
			Type(models.NetworkHTTP).
			Domains(s.containerHttpURL().Hostname(), "bacalhau.org").
			BuildOrDie()).
		Engine(s.curlTask()).
		BuildOrDie()

	result, err := s.runJob(task)
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingWithSubdomains() {
	s.T().Skip("subdomains fail domain validation")
	hostname := s.containerHttpURL().Hostname()
	hostroot := strings.Join(strings.SplitN(hostname, ".", 2)[:1], ".")

	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().
			Type(models.NetworkHTTP).
			Domains(hostname, hostroot).
			BuildOrDie()).
		Engine(s.curlTask()).
		BuildOrDie()

	result, err := s.runJob(task)
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingFiltersHTTP() {
	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().
			Type(models.NetworkHTTP).
			Domains("bacalhau.org").
			BuildOrDie()).
		Engine(s.curlTask()).
		BuildOrDie()

	result, err := s.runJob(task)
	// The curl will succeed but should return a non-zero exit code and error page.
	require.NoError(s.T(), err)
	require.NotZero(s.T(), result.ExitCode)
	require.Contains(s.T(), result.STDOUT, "ERROR: The requested URL could not be retrieved")
}

func (s *ExecutorTestSuite) TestDockerNetworkingFiltersHTTPS() {
	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().
			Type(models.NetworkHTTP).
			Domains(s.containerHttpURL().Hostname()).
			BuildOrDie()).
		Engine(dockermodels.NewDockerEngineBuilder(CurlDockerImage).
			WithEntrypoint("curl", "--fail-with-body", "https://www.bacalhau.org").
			Build()).
		BuildOrDie()

	result, err := s.runJob(task)

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
	task := mock.TaskBuilder().
		Network(models.NewNetworkConfigBuilder().Type(models.NetworkHTTP).Domains(s.containerHttpURL().Hostname()).BuildOrDie()).
		Engine(s.curlTask()).
		BuildOrDie()

	result, err := s.runJob(task)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "test", result.STDOUT, result.STDOUT)
}

func (s *ExecutorTestSuite) TestTimesOutCorrectly() {
	expected := "message after sleep"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	task := mock.TaskBuilder().
		Engine(
			dockermodels.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint("bash", "-c", fmt.Sprintf(`sleep 1 && echo "%s" && sleep 20`, expected)).
				Build()).
		BuildOrDie()

	result, err := s.runJobWithContext(ctx, task, "timeout")
	// The Docker client has changed so that it prioritizes container error message
	// and not the error message from the context. It does error upon timeout, but not
	// with a context.DeadlineExceeded error.
	s.Error(err)
	s.Truef(strings.HasPrefix(result.STDOUT, expected), "'%s' does not start with '%s'", result.STDOUT, expected)
}

func (s *ExecutorTestSuite) TestDockerStreamsAlreadyComplete() {
	id := "streams-fail"
	done := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	task := mock.TaskBuilder().
		Engine(
			dockermodels.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint("bash", "cat /sys/fs/cgroup/cpu.max").
				Build()).
		ResourcesConfig(models.NewResourcesConfigBuilder().CPU(CPU_LIMIT).Memory(MEMORY_LIMIT).BuildOrDie()).
		BuildOrDie()

	go func() {
		_, _ = s.runJobWithContext(ctx, task, id)
		done <- true
	}()

	reader, err := s.executor.GetOutputStream(ctx, id, true, true)

	<-done
	require.Nil(s.T(), reader)
	require.Error(s.T(), err)
}

func (s *ExecutorTestSuite) TestDockerStreamsSlowTask() {
	id := "streams-ok"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	task := mock.TaskBuilder().
		Engine(
			dockermodels.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint("bash", "-c", "echo hello && sleep 20").
				Build()).
		ResourcesConfig(models.NewResourcesConfigBuilder().CPU(CPU_LIMIT).Memory(MEMORY_LIMIT).BuildOrDie()).
		BuildOrDie()

	go func() {
		_, _ = s.runJobWithContext(ctx, task, id)
	}()

	// Give docker time to start the container, otherwise there
	// be nothing to retrieve the output from.
	time.Sleep(time.Duration(500) * time.Millisecond)

	reader, err := s.executor.GetOutputStream(ctx, id, true, true)

	require.NotNil(s.T(), reader)
	require.NoError(s.T(), err)

	df, err := logger.NewDataFrameFromReader(reader)
	require.NoError(s.T(), err)
	require.Equal(s.T(), string(df.Data), "hello\n")
	require.Equal(s.T(), df.Size, 6)
	require.Equal(s.T(), df.Tag, logger.StdoutStreamTag)
}
