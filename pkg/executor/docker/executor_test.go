//go:build integration || !unit

package docker

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
	ctx      context.Context
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}

func (s *ExecutorTestSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	docker.MustHaveDocker(s.T())
	s.ctx = context.Background()

	var err error

	cfg, err := config.NewTestConfig()
	require.NoError(s.T(), err)

	s.executor, err = NewExecutor(ExecutorParams{
		ID:     "bacalhau-executor-unit-test",
		Config: cfg.Engines.Types.Docker,
	})
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
		gateway, err = s.executor.client.HostGatewayIP(s.ctx)
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
	go s.server.Serve(listener)
}

// tearDownTest is called after each test in the suite.
func (s *ExecutorTestSuite) TearDownTest() {
	if s.server != nil {
		s.server.Close()
	}
	if s.executor != nil {
		s.executor.Shutdown(context.Background())
	}
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
	es, err := dockermodels.NewDockerEngineBuilder(CurlDockerImage).
		WithEntrypoint("curl", "--fail-with-body", s.containerHttpURL().JoinPath("hello.txt").String()).
		Build()
	s.Require().NoError(err)
	return es
}

func (s *ExecutorTestSuite) startJob(spec *models.Task, name string) {
	resultsPath, _ := compute.NewResultsPath(s.T().TempDir())
	executionDir, _ := resultsPath.PrepareExecutionOutputDir(name)
	j := mock.Job()
	j.ID = name
	j.Tasks = []*models.Task{spec}

	resources, err := spec.ResourcesConfig.ToResources()
	require.NoError(s.T(), err)

	s.Require().NoError(s.executor.Start(
		s.ctx,
		&executor.RunCommandRequest{
			JobID:        j.ID,
			ExecutionID:  name,
			Resources:    resources,
			Network:      spec.Network,
			Outputs:      spec.ResultPaths,
			Inputs:       nil,
			ExecutionDir: executionDir,
			EngineParams: spec.Engine,
			Env:          models.EnvVarsToStringMap(spec.Env),
			OutputLimits: executor.OutputLimits{
				MaxStdoutFileLength:   system.MaxStdoutFileLength,
				MaxStdoutReturnLength: system.MaxStdoutReturnLength,
				MaxStderrFileLength:   system.MaxStderrFileLength,
				MaxStderrReturnLength: system.MaxStderrReturnLength,
			},
		},
	))
}

func (s *ExecutorTestSuite) startJobAndWaitCompletion(spec *models.Task, name string) (*models.RunCommandResult, error) {
	s.startJob(spec, name)
	resultC, errC := s.executor.Wait(s.ctx, name)
	select {
	case out := <-resultC:
		return out, nil
	case err := <-errC:
		return nil, err
	}
}

// startJobAndWaitRunning starts a job and waits for it to be in the running state.
func (s *ExecutorTestSuite) startJobAndWaitRunning(spec *models.Task, name string) {
	s.startJob(spec, name)
	s.Eventuallyf(func() bool {
		c, err := s.executor.FindRunningContainer(s.ctx, name)
		if err == nil {
			s.T().Logf("found running container: %s for execution %s", c, name)
			return true
		}
		return false
	}, time.Second*5, time.Millisecond*50, "Container %s not running", name)
}

const (
	CPU_LIMIT = "100m"
	// 100 mebibytes is 104,857,600 bytes
	MEBIBYTE_MEMORY_LIMIT = "100MiB"
	// 100 megabytes is 100,000,000 bytes
	MEGABYTE_MEMORY_LIMIT = "100MB"

	CPU_LIMIT_UNITS = 0.1
	// 104,857,600 bytes
	MEBIBYTE_MEMORY_LIMIT_BYTES = 100 * 1024 * 1024
	// 100,000,000 bytes
	MEGABYTE_MEMORY_LIMIT_BYTES = 100 * 1000 * 1000
)

func (s *ExecutorTestSuite) TestDockerResourceLimitsCPU() {
	if runtime.GOOS == "windows" {
		s.T().Skip("Resource limits don't apply to containers running on Windows")
	}

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37").
		WithEntrypoint("sh", "-c", "cat /sys/fs/cgroup/cpu.max").
		Build()
	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es
	task.ResourcesConfig = &models.ResourcesConfig{CPU: CPU_LIMIT, Memory: MEBIBYTE_MEMORY_LIMIT}
	task.Normalize()

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	require.NoError(s.T(), err)

	values := strings.Fields(result.STDOUT)
	s.Require().Len(values, 2, "the container reported CPU (%s) does not match the expected format", result)

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

	tests := []struct {
		in  string
		exp int
	}{
		{MEGABYTE_MEMORY_LIMIT, MEGABYTE_MEMORY_LIMIT_BYTES},
		{MEBIBYTE_MEMORY_LIMIT, MEBIBYTE_MEMORY_LIMIT_BYTES},
	}

	for _, p := range tests {
		es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37").
			WithEntrypoint("sh", "-c", "cat /sys/fs/cgroup/memory.max").
			Build()
		s.Require().NoError(err)

		task := mock.Task()
		task.Engine = es
		task.ResourcesConfig = &models.ResourcesConfig{CPU: CPU_LIMIT, Memory: p.in}
		task.Normalize()

		result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
		require.NoError(s.T(), err)

		s.Require().NotEmpty(result.STDOUT, "the container reported memory returned an empty string")

		intVar, err := strconv.Atoi(strings.TrimSpace(result.STDOUT))
		require.NoError(s.T(), err)

		// Docker adjusts the memory limit to align with the Linux kernel's memory management,
		// which works at the granularity of a memory page size (generally 4096 bytes or 4KiB).
		// When setting the memory limit, Docker will round down to an even division of the page size.
		// Therefore, this test checks if the absolute difference between the actual memory limit inside the container
		// and the expected memory limit is less than or equal to one memory page size (4096 bytes or 4KiB).
		// This means that even with the rounding down, the memory limit inside the Docker container does not exceed our limit by more than one page size.
		diff := int(math.Abs(float64(intVar - p.exp)))
		require.LessOrEqual(s.T(), diff, 4096, "the difference between the container reported memory and the configured limit exceeds the page size")
	}
}

func (s *ExecutorTestSuite) TestDockerNetworkingFull() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHost}

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingNone() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkNone}

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	require.NoError(s.T(), err)
	require.Empty(s.T(), result.STDOUT)
	require.NotEmpty(s.T(), result.STDERR)
	require.NotZero(s.T(), result.ExitCode)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTP() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{s.containerHttpURL().Hostname()}}

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTPWithMultipleDomains() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{
		s.containerHttpURL().Hostname(), "bacalhau.org"}}

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingWithSubdomains() {
	s.T().Skip("subdomains fail domain validation")
	hostname := s.containerHttpURL().Hostname()
	hostroot := strings.Join(strings.SplitN(hostname, ".", 2)[:1], ".")

	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{hostname, hostroot}}

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingFiltersHTTP() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{"bacalhau.org"}}

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	// The curl will succeed but should return a non-zero exit code and error page.
	require.NoError(s.T(), err)
	require.NotZero(s.T(), result.ExitCode)
	require.Contains(s.T(), result.STDOUT, "ERROR: The requested URL could not be retrieved")
}

func (s *ExecutorTestSuite) TestDockerNetworkingFiltersHTTPS() {
	es, err := dockermodels.NewDockerEngineBuilder(CurlDockerImage).
		WithEntrypoint("curl", "--fail-with-body", "https://www.bacalhau.org").
		Build()
	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{s.containerHttpURL().Hostname()}}

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())

	// The curl will succeed but should return a non-zero exit code and error page.
	require.NoError(s.T(), err)
	require.NotZero(s.T(), result.ExitCode)
	require.Empty(s.T(), result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerExecutionCancellation() {
	resultC := make(chan *models.RunCommandResult, 1)
	errC := make(chan error, 1)
	executionID := uuid.New().String()

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", "sleep 30").
		Build()

	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es

	go func() {
		result, err := s.startJobAndWaitCompletion(task, executionID)
		if err != nil {
			errC <- err
		} else {
			resultC <- result
		}
	}()

	s.Require().Eventually(func() bool {
		handler, ok := s.executor.handlers.Get(executionID)
		return ok && handler.active()
	}, time.Second*10, time.Millisecond*100, "Could not find a running container")

	// This is important to do. In our docker executor, we set active to true, before calling the docker client with ContainerStart
	// Hence there is a bit of time before the container actually gets started. The correct way of identifying that whether
	// a container has started or not is via activeCh. We want to make sure that container is started before canceling the execution.
	handler, _ := s.executor.handlers.Get(executionID)
	<-handler.activeCh

	err = s.executor.Cancel(s.ctx, executionID)
	s.Require().NoError(err)

	select {
	case err := <-errC:
		s.Require().Failf("Executor run should have returned a result, but instead returned err: %w", err.Error())
	case result := <-resultC:
		s.Require().NotNil(result)
	}
}

func (s *ExecutorTestSuite) TestDockerNetworkingAppendsHTTPHeader() {
	s.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(r.Header.Get("X-Bacalhau-Job-ID")))
		s.Require().NoError(err)
	})

	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{s.containerHttpURL().Hostname()}}

	executionID := uuid.New().String()
	result, err := s.startJobAndWaitCompletion(task, executionID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), executionID, result.STDOUT, result.STDOUT)
}

func (s *ExecutorTestSuite) TestTimesOutCorrectly() {
	expected := "message after sleep"
	var cancel context.CancelFunc
	s.ctx, cancel = context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", fmt.Sprintf(`sleep 1 && echo "%s" && sleep 20`, expected)).
		Build()
	s.Require().NoError(err)
	task := mock.Task()
	task.Engine = es

	name := "timeout"
	resultsPath, _ := compute.NewResultsPath(s.T().TempDir())
	executionDir, _ := resultsPath.PrepareExecutionOutputDir(name)
	j := mock.Job()
	j.ID = name
	j.Tasks = []*models.Task{task}

	resources, err := task.ResourcesConfig.ToResources()
	require.NoError(s.T(), err)

	s.Require().NoError(s.executor.Start(s.ctx,
		&executor.RunCommandRequest{
			JobID:        j.ID,
			ExecutionID:  name,
			Resources:    resources,
			Network:      task.Network,
			Outputs:      task.ResultPaths,
			Inputs:       nil,
			ExecutionDir: executionDir,
			EngineParams: task.Engine,
			OutputLimits: executor.OutputLimits{
				MaxStdoutFileLength:   system.MaxStdoutFileLength,
				MaxStdoutReturnLength: system.MaxStdoutReturnLength,
				MaxStderrFileLength:   system.MaxStderrFileLength,
				MaxStderrReturnLength: system.MaxStderrReturnLength,
			},
		},
	))

	ticker := time.NewTimer(time.Second * 10)
	// use a different context for waiting as we don't want to timeout waiting on the job.
	resC, errC := s.executor.Wait(context.Background(), name)
	select {
	case res := <-resC:
		// we expect to receive an error from the executions result stating the deadline for
		// execution was exceeded.
		s.Require().Contains(res.ErrorMsg, context.DeadlineExceeded.Error())
	case err := <-errC:
		s.T().Fatal(err)
	case <-ticker.C:
		s.T().Fatal("container was not canceled.")
	}
}

func (s *ExecutorTestSuite) TestDockerStreamsAlreadyComplete() {
	id := "streams-exited-container"

	expectedOutput := "some job output"
	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", fmt.Sprintf("echo %s", expectedOutput)).
		Build()
	s.Require().NoError(err)
	task := mock.Task()
	task.Engine = es
	task.ResourcesConfig = &models.ResourcesConfig{CPU: CPU_LIMIT, Memory: MEBIBYTE_MEMORY_LIMIT}
	task.Normalize()

	_, err = s.startJobAndWaitCompletion(task, id)
	s.Require().NoError(err)

	reader, err := s.executor.GetLogStream(s.ctx, messages.ExecutionLogsRequest{
		ExecutionID: id,
		Tail:        false,
		Follow:      false,
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), reader)

	ch := logstream.NewLiveStreamer(logstream.LiveStreamerParams{
		Reader: reader,
	}).Stream(s.ctx)
	res, ok := <-ch
	require.True(s.T(), ok)
	executionLog := res.Value
	require.Equal(s.T(), string(executionLog.Line), expectedOutput+"\n") // LiveStreamer adds line break
	require.Equal(s.T(), executionLog.Type, models.ExecutionLogTypeSTDOUT)
}

func (s *ExecutorTestSuite) TestDockerStreamsSlowTask() {
	id := "streams-ok"
	sleepSeconds := 3

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", fmt.Sprintf("echo hello && sleep %d", sleepSeconds)).
		Build()
	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es
	task.ResourcesConfig = &models.ResourcesConfig{CPU: CPU_LIMIT, Memory: MEBIBYTE_MEMORY_LIMIT}
	task.Normalize()
	s.startJobAndWaitRunning(task, id)

	reader, err := s.executor.GetLogStream(s.ctx, messages.ExecutionLogsRequest{
		ExecutionID: id,
		Tail:        false,
		Follow:      true,
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), reader)

	ch := logstream.NewLiveStreamer(logstream.LiveStreamerParams{
		Reader: reader,
	}).Stream(s.ctx)
	res, ok := <-ch
	require.True(s.T(), ok)
	executionLog := res.Value
	require.Equal(s.T(), string(executionLog.Line), "hello\n")
	require.Equal(s.T(), executionLog.Type, models.ExecutionLogTypeSTDOUT)

	// verify the channel returns when the job is completed after sleepSeconds
	// if test becomes flaky, then consider increasing sleepSeconds or remove this validation
	_, ok = <-ch
	require.False(s.T(), ok)
}

func (s *ExecutorTestSuite) TestDockerOOM() {
	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("tail", "/dev/zero").
		Build()
	s.Require().NoError(err)
	task := mock.Task()
	task.Engine = es

	result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
	require.NoError(s.T(), err)
	require.Contains(s.T(), result.ErrorMsg, "memory limit exceeded")
}

func (s *ExecutorTestSuite) TestDockerEnvironmentVariables() {
	tests := []struct {
		name      string
		taskEnv   map[string]models.EnvVarValue
		engineEnv []string
		checkVars []string // variables to check in order
		want      string
	}{
		{
			name: "task environment variables",
			taskEnv: map[string]models.EnvVarValue{
				"TEST_VAR":    "test_value",
				"ANOTHER_VAR": "another_value",
			},
			checkVars: []string{"TEST_VAR", "ANOTHER_VAR"},
			want:      "test_value\nanother_value",
		},
		{
			name: "engine environment variables",
			engineEnv: []string{
				"TEST_VAR=engine_value",
				"ENGINE_VAR=engine_only",
			},
			checkVars: []string{"TEST_VAR", "ENGINE_VAR"},
			want:      "engine_value\nengine_only",
		},
		{
			name: "merged environment variables with engine precedence",
			taskEnv: map[string]models.EnvVarValue{
				"TEST_VAR": "task_value",
				"TASK_VAR": "task_only",
			},
			engineEnv: []string{
				"TEST_VAR=engine_value",
				"ENGINE_VAR=engine_only",
			},
			checkVars: []string{"TEST_VAR", "TASK_VAR", "ENGINE_VAR"},
			want:      "engine_value\ntask_only\nengine_only",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create simple script that prints vars in order
			script := strings.Builder{}
			for _, v := range tt.checkVars {
				script.WriteString(fmt.Sprintf("echo $%s\n", v))
			}

			builder := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
				WithEntrypoint("sh", "-c", script.String())

			if len(tt.engineEnv) > 0 {
				builder = builder.WithEnvironmentVariables(tt.engineEnv...)
			}

			es, err := builder.Build()
			s.Require().NoError(err)

			task := mock.Task()
			task.Engine = es
			task.Env = tt.taskEnv

			result, err := s.startJobAndWaitCompletion(task, uuid.New().String())
			require.NoError(s.T(), err)
			require.Zero(s.T(), result.ExitCode, result.STDERR)

			output := strings.TrimSpace(result.STDOUT)
			s.Equal(tt.want, output)
		})
	}
}

func (s *ExecutorTestSuite) TestPortMappingInHostMode() {
	testutils.SkipIfNotLinux(s.T(), "docker host mode is not supported on non-linux platforms")

	port, err := network.GetFreePort()
	s.Require().NoError(err)

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", fmt.Sprintf(`while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p %d; done`, port)).
		Build()
	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es
	task.Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: models.PortMap{
			{
				Name:   "http",
				Static: port,
			},
		},
	}

	// Start the container
	s.startJob(task, "host-network-test")

	// In host mode, the container port should be directly accessible on the host
	s.Require().Eventually(func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *ExecutorTestSuite) TestPortMappingInBridgeMode() {
	// Create a container that listens on the target port
	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", `while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p 80; done`).
		Build()
	s.Require().NoError(err)

	port, err := network.GetFreePort()
	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es
	task.Network = &models.NetworkConfig{
		Type: models.NetworkBridge,
		Ports: models.PortMap{
			{
				Name:   "http",
				Static: port, // Host port
				Target: 80,   // Container port
			},
		},
	}

	// Start the container
	s.startJob(task, "bridge-network-test")

	// In bridge mode, the container port should be accessible via the mapped host port
	s.Require().Eventually(func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *ExecutorTestSuite) TestMultiplePortMappingsInBridgeMode() {
	// Create a container that listens on both ports
	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", `
			(while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p 80; done) &
			(while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p 81; done)
		`).
		Build()
	s.Require().NoError(err)

	port1, err := network.GetFreePort()
	s.Require().NoError(err)

	port2, err := network.GetFreePort()
	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es
	task.Network = &models.NetworkConfig{
		Type: models.NetworkBridge,
		Ports: models.PortMap{
			{
				Name:   "http1",
				Static: port1,
				Target: 80,
			},
			{
				Name:   "http2",
				Static: port2,
				Target: 81,
			},
		},
	}

	// Start the container
	s.startJob(task, "bridge-network-multiple-ports-test")

	// Both ports should be accessible
	s.Require().Eventually(func() bool {
		resp1, err := http.Get(fmt.Sprintf("http://localhost:%d", port1))
		if err != nil {
			return false
		}
		defer func() { _ = resp1.Body.Close() }()

		resp2, err := http.Get(fmt.Sprintf("http://localhost:%d", port2))
		if err != nil {
			return false
		}
		defer func() { _ = resp2.Body.Close() }()

		return resp1.StatusCode == http.StatusOK && resp2.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *ExecutorTestSuite) TestAccessToHostService() {
	// Start a simple HTTP server on the host
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("host-service"))
	})

	listener, err := net.Listen("tcp", ":0") // Let OS choose port
	s.Require().NoError(err)
	defer func() { _ = listener.Close() }()

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer func() { _ = server.Close() }()

	// Get the port that was assigned
	hostPort := listener.Addr().(*net.TCPAddr).Port

	tests := []struct {
		name        string
		networkType models.Network
		hostURL     string
	}{
		{
			name:        "host network mode",
			networkType: models.NetworkHost,
			hostURL:     fmt.Sprintf("http://localhost:%d", hostPort),
		},
		{
			name:        "bridge network mode",
			networkType: models.NetworkBridge,
			hostURL:     fmt.Sprintf("http://host.docker.internal:%d", hostPort),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.networkType == models.NetworkHost {
				testutils.SkipIfNotLinux(s.T(), "docker host mode is not supported on non-linux platforms")
			}

			// Create a task that tries to connect to our test server
			es, err := dockermodels.NewDockerEngineBuilder(CurlDockerImage).
				WithEntrypoint("curl", "-s", tt.hostURL).
				Build()
			s.Require().NoError(err)

			task := mock.Task()
			task.Engine = es
			task.Network = &models.NetworkConfig{
				Type: tt.networkType,
			}

			result, err := s.startJobAndWaitCompletion(task, fmt.Sprintf("access-host-%s", tt.networkType))
			s.Require().NoError(err)
			s.Equal("host-service", result.STDOUT)
		})
	}
}
