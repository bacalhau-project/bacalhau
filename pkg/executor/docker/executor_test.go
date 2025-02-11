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

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
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
	logger.ConfigureTestLogging(s.T())
	docker.MustHaveDocker(s.T())

	var err error
	s.cm = system.NewCleanupManager()
	s.T().Cleanup(func() {
		s.cm.Cleanup(context.Background())
	})

	cfg, err := config.NewTestConfig()
	require.NoError(s.T(), err)

	s.executor, err = NewExecutor("bacalhau-executor-unit-test", cfg.Engines.Types.Docker)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() {
		if err := s.executor.Shutdown(context.Background()); err != nil {
			s.T().Logf("failed to shutdown executor: %s", err)
		}
	})

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
	es, err := dockermodels.NewDockerEngineBuilder(CurlDockerImage).
		WithEntrypoint("curl", "--fail-with-body", s.containerHttpURL().JoinPath("hello.txt").String()).
		Build()
	s.Require().NoError(err)
	return es
}

func (s *ExecutorTestSuite) runJob(spec *models.Task, executionID string) (*models.RunCommandResult, error) {
	return s.runJobWithContext(context.Background(), spec, executionID)
}

func (s *ExecutorTestSuite) startJob(spec *models.Task, name string) {
	result := s.T().TempDir()
	j := mock.Job()
	j.ID = name
	j.Tasks = []*models.Task{spec}

	resources, err := spec.ResourcesConfig.ToResources()
	require.NoError(s.T(), err)

	s.Require().NoError(s.executor.Start(
		context.Background(),
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
	))
}

func (s *ExecutorTestSuite) runJobWithContext(ctx context.Context, spec *models.Task, name string) (*models.RunCommandResult, error) {
	result := s.T().TempDir()
	j := mock.Job()
	j.ID = name
	j.Tasks = []*models.Task{spec}

	resources, err := spec.ResourcesConfig.ToResources()
	require.NoError(s.T(), err)

	s.Require().NoError(s.executor.Start(
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
			Env:          models.EnvVarsToStringMap(spec.Env),
			OutputLimits: executor.OutputLimits{
				MaxStdoutFileLength:   system.MaxStdoutFileLength,
				MaxStdoutReturnLength: system.MaxStdoutReturnLength,
				MaxStderrFileLength:   system.MaxStderrFileLength,
				MaxStderrReturnLength: system.MaxStderrReturnLength,
			},
		},
	))
	resultC, errC := s.executor.Wait(ctx, name)
	select {
	case out := <-resultC:
		return out, nil
	case err := <-errC:
		return nil, err
	}
}

func (s *ExecutorTestSuite) runJobGetStdout(spec *models.Task, executionID string) (string, error) {
	runnerOutput, runErr := s.runJob(spec, executionID)
	return runnerOutput.STDOUT, runErr
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

	result, err := s.runJobGetStdout(task, uuid.New().String())
	require.NoError(s.T(), err)

	values := strings.Fields(result)
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

		result, err := s.runJobGetStdout(task, uuid.New().String())
		require.NoError(s.T(), err)

		s.Require().NotEmpty(result, "the container reported memory returned an empty string")

		intVar, err := strconv.Atoi(strings.TrimSpace(result))
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
	task.Network = &models.NetworkConfig{Type: models.NetworkFull}

	result, err := s.runJob(task, uuid.New().String())
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingNone() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkNone}

	result, err := s.runJob(task, uuid.New().String())
	require.NoError(s.T(), err)
	require.Empty(s.T(), result.STDOUT)
	require.NotEmpty(s.T(), result.STDERR)
	require.NotZero(s.T(), result.ExitCode)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTP() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{s.containerHttpURL().Hostname()}}

	result, err := s.runJob(task, uuid.New().String())
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingHTTPWithMultipleDomains() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{
		s.containerHttpURL().Hostname(), "bacalhau.org"}}

	result, err := s.runJob(task, uuid.New().String())
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

	result, err := s.runJob(task, uuid.New().String())
	require.NoError(s.T(), err, result.STDERR)
	require.Zero(s.T(), result.ExitCode, result.STDERR)
	require.Equal(s.T(), "/hello.txt", result.STDOUT)
}

func (s *ExecutorTestSuite) TestDockerNetworkingFiltersHTTP() {
	task := mock.Task()
	task.Engine = s.curlTask()
	task.Network = &models.NetworkConfig{Type: models.NetworkHTTP, Domains: []string{"bacalhau.org"}}

	result, err := s.runJob(task, uuid.New().String())
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

	result, err := s.runJob(task, uuid.New().String())

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

	jobCtx := context.Background()
	go func() {
		result, err := s.runJobWithContext(jobCtx, task, executionID)
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

	err = s.executor.Cancel(jobCtx, executionID)
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
	result, err := s.runJob(task, executionID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), executionID, result.STDOUT, result.STDOUT)
}

func (s *ExecutorTestSuite) TestTimesOutCorrectly() {
	expected := "message after sleep"
	jobCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", fmt.Sprintf(`sleep 1 && echo "%s" && sleep 20`, expected)).
		Build()
	s.Require().NoError(err)
	task := mock.Task()
	task.Engine = es

	name := "timeout"
	resultDir := s.T().TempDir()
	j := mock.Job()
	j.ID = name
	j.Tasks = []*models.Task{task}

	resources, err := task.ResourcesConfig.ToResources()
	require.NoError(s.T(), err)

	s.Require().NoError(s.executor.Start(jobCtx,
		&executor.RunCommandRequest{
			JobID:        j.ID,
			ExecutionID:  name,
			Resources:    resources,
			Network:      task.Network,
			Outputs:      task.ResultPaths,
			Inputs:       nil,
			ResultsDir:   resultDir,
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
	id := "streams-fail"
	done := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "cat /sys/fs/cgroup/cpu.max").
		Build()
	s.Require().NoError(err)
	task := mock.Task()
	task.Engine = es
	task.ResourcesConfig = &models.ResourcesConfig{CPU: CPU_LIMIT, Memory: MEBIBYTE_MEMORY_LIMIT}
	task.Normalize()

	go func() {
		_, _ = s.runJobWithContext(ctx, task, id)
		done <- true
	}()
	<-done
	reader, err := s.executor.GetLogStream(ctx, messages.ExecutionLogsRequest{
		ExecutionID: id,
		Tail:        true,
		Follow:      true,
	})

	require.Nil(s.T(), reader)
	require.Error(s.T(), err)
}

func (s *ExecutorTestSuite) TestDockerStreamsSlowTask() {
	id := "streams-ok"

	es, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", "echo hello && sleep 20").
		Build()
	s.Require().NoError(err)

	task := mock.Task()
	task.Engine = es
	task.ResourcesConfig = &models.ResourcesConfig{CPU: CPU_LIMIT, Memory: MEBIBYTE_MEMORY_LIMIT}
	task.Normalize()
	s.startJob(task, id)

	reader, err := s.executor.GetLogStream(context.Background(), messages.ExecutionLogsRequest{
		ExecutionID: id,
		Tail:        true,
		Follow:      true,
	})

	require.NotNil(s.T(), reader)
	require.NoError(s.T(), err)

	ch := logstream.NewLiveStreamer(logstream.LiveStreamerParams{
		Reader: reader,
	}).Stream(context.Background())
	res, ok := <-ch
	require.True(s.T(), ok)
	executionLog := res.Value
	require.Equal(s.T(), string(executionLog.Line), "hello\n")
	require.Equal(s.T(), executionLog.Type, models.ExecutionLogTypeSTDOUT)

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

	result, err := s.runJob(task, uuid.New().String())
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

			result, err := s.runJob(task, uuid.New().String())
			require.NoError(s.T(), err)
			require.Zero(s.T(), result.ExitCode, result.STDERR)

			output := strings.TrimSpace(result.STDOUT)
			s.Equal(tt.want, output)
		})
	}
}
