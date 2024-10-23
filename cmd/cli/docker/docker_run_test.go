//go:build integration || !unit

package docker_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/google/uuid"
	"sigs.k8s.io/yaml"

	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	storage_url "github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/docker"

	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DockerRunSuite struct {
	cmdtesting.BaseSuite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDockerRunSuite(t *testing.T) {
	suite.Run(t, new(DockerRunSuite))
}

// Before each test
func (s *DockerRunSuite) SetupTest() {
	docker.MustHaveDocker(s.T())
	s.BaseSuite.SetupTest()
}

func (s *DockerRunSuite) TestRun_GenericSubmit() {
	ctx := context.Background()
	randomUUID := uuid.New()
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"busybox:latest",
		"echo",
		randomUUID.String(),
	)
	s.Require().NoError(err, "failed to submit job")

	testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
}

func (s *DockerRunSuite) TestRun_DryRun() {
	randomUUID := uuid.New()
	entrypointCommand := fmt.Sprintf("echo %s", randomUUID.String())
	stdout, _, err := s.Execute("docker", "run",
		"busybox:latest",
		entrypointCommand,
		"--dry-run",
	)
	s.Require().NoError(err, "Error submitting job.")

	s.Require().Contains(stdout, randomUUID.String(), "Dry run failed to contain UUID %s", randomUUID.String())

	var j *models.Job
	s.Require().NoError(yaml.Unmarshal([]byte(stdout), &j))
	s.Require().NotNil(j, "Failed to unmarshal job from dry run output")

	dockerSpec, err := dockermodels.DecodeSpec(j.Task().Engine)
	s.Require().NoError(err)
	s.Require().Equal(entrypointCommand, dockerSpec.Parameters[0], "Dry run job should not have an ID")
}

func (s *DockerRunSuite) TestRun_GPURequests() {
	if !s.Node.ComputeNode.Capacity.IsWithinLimits(context.Background(), models.Resources{GPU: 1}) {
		s.T().Skip("Skipping test as no GPU is available in current host")
	}
	tests := []struct {
		submitArgs []string
		fatalErr   bool
		errString  string
		numGPUs    string
	}{
		{submitArgs: []string{"--gpu=1", "nvidia/cuda:11.0.3-base-ubuntu20.04", "nvidia-smi"}, fatalErr: false, errString: "", numGPUs: "1"},
	}

	for i, tc := range tests {
		func() {
			var logBuf = new(bytes.Buffer)
			var Stdout = struct{ io.Writer }{os.Stdout}
			originalLogger := log.Logger
			log.Logger = log.With().Logger().Output(io.MultiWriter(Stdout, logBuf))
			defer func() {
				log.Logger = originalLogger
			}()

			ctx := context.Background()
			allArgs := []string{"docker", "run"}
			allArgs = append(allArgs, tc.submitArgs...)
			_, out, submitErr := s.ExecuteTestCobraCommand(allArgs...)

			if tc.fatalErr {
				s.Require().Contains(out, tc.errString, "Did not find expected error message for fatalError in error string.\nExpected: %s\nActual: %s", tc.errString, out)
				return
			} else {
				s.Require().NoError(submitErr, "Error submitting job. Run - Test-Number: %d - String: %s", i, tc.submitArgs)
			}

			s.Require().True(!tc.fatalErr, "Expected fatal err, but submitted.")

			j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

			if tc.errString != "" {
				o := logBuf.String()
				s.Require().Contains(o, tc.errString, "Did not find expected error message in error string.\nExpected: %s\nActual: %s", tc.errString, o)
			}
			s.Require().Equal(tc.numGPUs, j.Task().ResourcesConfig.GPU, "Expected %d GPUs, but got %d", tc.numGPUs, j.Task().ResourcesConfig.GPU)
		}()
	}
}

func (s *DockerRunSuite) TestRun_GenericSubmitWait() {
	ctx := context.Background()

	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"--wait",
		"busybox:latest",
		"--",
		"echo", "hello from docker submit wait",
	)
	s.Require().NoErrorf(err, "Error submitting job.")

	_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
}

func (s *DockerRunSuite) TestRun_SubmitUrlInputs() {
	type (
		InputURL struct {
			url             string
			pathInContainer string
			flag            string
			filename        string
		}
	)

	// For URLs, the input should be a file, the output a directory
	// Internally the URL storage provider appends the filename to the directory path
	testURLs := []struct {
		inputURL InputURL
	}{
		{inputURL: InputURL{url: "https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/README.md", pathInContainer: "/inputs", filename: "README.md", flag: "-i"}},
		{inputURL: InputURL{url: "https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/main.go", pathInContainer: "/inputs", filename: "main.go", flag: "-i"}},
	}

	for _, urls := range testURLs {
		ctx := context.Background()
		flagsArray := []string{"docker", "run"}

		flagsArray = append(flagsArray, urls.inputURL.flag, urls.inputURL.url)
		flagsArray = append(flagsArray, "busybox:latest", "cat", fmt.Sprintf("%s/%s", urls.inputURL.pathInContainer, urls.inputURL.filename))

		_, out, err := s.ExecuteTestCobraCommand(flagsArray...)
		s.Require().NoError(err, "Error submitting job")

		j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

		s.Require().Equal(1, len(j.Task().InputSources), "Number of job urls != # of test urls.")
		urlSpec, err := storage_url.DecodeSpec(j.Task().InputSources[0].Source)
		s.Require().NoError(err)
		s.Require().Equal(urls.inputURL.url, urlSpec.URL, "Test URL not equal to URL from job.")
		s.Require().Equal(urls.inputURL.pathInContainer, j.Task().InputSources[0].Target, "Test Path not equal to Path from job.")

	}
}

func (s *DockerRunSuite) TestRun_CreatedAt() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"busybox:latest",
		"echo", "'hello world'",
	)
	s.NoError(err, "Error submitting job.")

	j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

	s.Require().LessOrEqual(j.CreateTime, time.Now().UnixNano(), "Created at time is not less than or equal to now.")

	oldStartTime, _ := time.Parse(time.RFC3339, "2021-01-01T01:01:01+00:00")
	s.Require().GreaterOrEqual(j.CreateTime, oldStartTime.UnixNano(), "Created at time is not greater or equal to 2022-01-01.")
}

func (s *DockerRunSuite) TestRun_EdgeCaseCLI() {
	tests := []struct {
		submitArgs []string
		fatalErr   bool
		errString  string
	}{
		{submitArgs: []string{"busybox:latest", "-xoo -bar -baz"}, fatalErr: true, errString: "unknown shorthand flag"}, // submitting flag will fail if not separated with a --
		{submitArgs: []string{"busybox:latest", "python -xoo -bar -baz"}, fatalErr: false, errString: ""},               // separating with -- should work and allow flags
		// {submitString: "-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images -o results:/output_images dpokidov/imagemagick -- magick mogrify -fx '((g-b)/(r+g+b))>0.02 ? 1 : 0' -resize 256x256 -quality 100 -path /output_images /input_images/*.jpg"},
	}

	for i, tc := range tests {
		var logBuf = new(bytes.Buffer)
		var Stdout = struct{ io.Writer }{os.Stdout}
		originalLogger := log.Logger
		log.Logger = log.With().Logger().Output(io.MultiWriter(Stdout, logBuf))
		defer func() {
			log.Logger = originalLogger
		}()

		ctx := context.Background()
		allArgs := []string{"docker", "run"}
		allArgs = append(allArgs, tc.submitArgs...)
		_, out, submitErr := s.ExecuteTestCobraCommand(allArgs...)

		if tc.fatalErr {
			s.Require().Contains(out, tc.errString, "Did not find expected error message for fatalError in error string.\nExpected: %s\nActual: %s", tc.errString, out)
			return
		} else {
			s.Require().NoError(submitErr, "Error submitting job. Run - Test-Number: %d - String: %s", i, tc.submitArgs)
		}

		s.Require().True(!tc.fatalErr, "Expected fatal err, but submitted.")

		_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

		if tc.errString != "" {
			o := logBuf.String()
			s.Require().Contains(o, tc.errString, "Did not find expected error message in error string.\nExpected: %s\nActual: %s", tc.errString, o)
		}
	}
}

func (s *DockerRunSuite) TestRun_SubmitWorkdir() {
	tests := []struct {
		workdir   string
		errorCode int
	}{
		{workdir: "", errorCode: 0},
		{workdir: "/", errorCode: 0},
		{workdir: "./myDir", errorCode: 1},
		{workdir: "../myDir", errorCode: 1},
		{workdir: "http://foo.com", errorCode: 1},
		{workdir: "/foo//", errorCode: 0}, // double forward slash is allowed in unix
		{workdir: "/foo//bar", errorCode: 0},
	}

	for _, tc := range tests {
		ctx := context.Background()
		flagsArray := []string{"docker", "run"}
		flagsArray = append(flagsArray, "-w", tc.workdir)
		flagsArray = append(flagsArray, "busybox:latest", "pwd")

		_, out, err := s.ExecuteTestCobraCommand(flagsArray...)

		if tc.errorCode != 0 {
			s.Require().NotNil(err, "Expected fatal error, but none found")
		} else {
			s.Require().NoError(err, "Error submitting job.")

			j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

			dockerSpec, err := dockermodels.DecodeSpec(j.Task().Engine)
			s.Require().NoError(err)
			s.Require().Equal(tc.workdir, dockerSpec.WorkingDirectory, "Job workdir != test workdir.")
			s.Require().NoError(err, "Error in running command.")
		}
	}
}

func (s *DockerRunSuite) TestRun_ExplodeVideos() {
	videos := []string{
		"Bird flying over the lake.mp4",
		"Calm waves on a rocky sea gulf.mp4",
		"Prominent Late Gothic styled architecture.mp4",
	}

	for _, video := range videos {
		err := os.WriteFile(
			filepath.Join(s.AllowListedPath, video),
			[]byte(fmt.Sprintf("hello %s", video)),
			0644,
		)
		s.Require().NoError(err)
	}

	allArgs := []string{
		"docker", "run",
		"--wait",
		"-i", fmt.Sprintf("file://%s,dst=/inputs", s.AllowListedPath),
		"busybox:latest", "echo", "hello",
	}

	_, _, submitErr := s.ExecuteTestCobraCommand(allArgs...)
	s.Require().NoError(submitErr)
}

func (s *DockerRunSuite) TestTruncateReturn() {
	// Make it artificially small for this run
	oldStderrLength := system.MaxStderrReturnLength
	oldStdoutLength := system.MaxStdoutReturnLength
	system.MaxStderrReturnLength = 10
	system.MaxStdoutReturnLength = 10
	s.T().Cleanup(func() {
		system.MaxStderrReturnLength = oldStderrLength
		system.MaxStdoutReturnLength = oldStdoutLength
	})

	tests := map[string]struct {
		inputLength    datasize.ByteSize
		expectedLength datasize.ByteSize
		truncated      bool
	}{
		// "zero length": {inputLength: 0, truncated: false, expectedLength: 0},
		// "one length":  {inputLength: 1, truncated: false, expectedLength: 1},
		"maxLength - 1": {
			inputLength:    system.MaxStdoutReturnLength - 1,
			truncated:      false,
			expectedLength: system.MaxStdoutReturnLength - 1,
		},
		"maxLength": {
			inputLength:    system.MaxStdoutReturnLength,
			truncated:      false,
			expectedLength: system.MaxStdoutReturnLength,
		},
		"maxLength + 1": {
			inputLength:    system.MaxStdoutReturnLength + 1,
			truncated:      true,
			expectedLength: system.MaxStdoutReturnLength,
		},
		"maxLength + 10000": {
			inputLength:    system.MaxStdoutReturnLength * 10,
			truncated:      true,
			expectedLength: system.MaxStdoutReturnLength,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			ctx := context.Background()
			_, out, err := s.ExecuteTestCobraCommand(
				"docker", "run",
				"busybox:latest", "--", "perl", "-e", fmt.Sprintf(`print "=" x %d`, tc.inputLength),
			)
			s.Require().NoError(err, "Error submitting job. Name: %s. Expected Length: %s", name, tc.expectedLength)

			j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
			info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{
				JobID:   j.ID,
				Include: "executions",
			})
			s.Require().NoError(err)

			s.Len(info.Executions.Items, 1)
			s.Len(info.Executions.Items[0].RunOutput.STDOUT, int(tc.expectedLength.Bytes()))
		})
	}
}

func (s *DockerRunSuite) TestRun_MultipleURLs() {
	tests := []struct {
		expectedVolumes int
		inputFlags      []string
	}{
		{
			0,
			[]string{},
		},
		{
			1,
			[]string{"-i", "http://127.0.0.1/url1,dst=/inputs/url1.txt"},
		},
		{
			2,
			[]string{
				"-i", "http://127.0.0.1/url1.txt,dst=/inputs/url1.txt",
				"-i", "http://127.0.0.1/url2.txt,dst=/inputs/url2.txt",
			},
		},
	}

	for _, tc := range tests {
		s.Run(fmt.Sprintf("Expected Volumes %d", tc.expectedVolumes), func() {
			ctx := context.Background()
			var args []string

			args = append(args, "docker", "run")
			args = append(args, tc.inputFlags...)
			args = append(args, "busybox:latest", "--", "ls", "/input")

			_, out, err := s.ExecuteTestCobraCommand(args...)
			s.Require().NoError(err, "Error submitting job")

			j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

			s.Require().Equal(tc.expectedVolumes, len(j.Task().InputSources))
		})
	}
}

// Test bad images and bad binaries
func (s *DockerRunSuite) TestRun_BadExecutables() {
	tests := map[string]struct {
		imageName         string
		executable        string
		isValid           bool
		errStringContains string
	}{
		"good-image-good-executable": {
			imageName:         "busybox:latest", // Good image // TODO we consider an untagged image poor practice, fix this
			executable:        "ls",             // Good executable
			isValid:           true,
			errStringContains: "",
		},
		"bad-image-good-executable": {
			imageName:         "badimage", // Bad image
			executable:        "ls",       // Good executable
			isValid:           false,
			errStringContains: "image not available",
		},
		"good-image-bad-executable": {
			imageName:         "busybox:latest", // Good image // TODO we consider an untagged image poor practice, fix this
			executable:        "BADEXECUTABLE",  // Bad executable
			isValid:           false,
			errStringContains: "executable file not found",
		},
		"bad-image-bad-executable": {
			imageName:         "badimage",      // Bad image
			executable:        "BADEXECUTABLE", // Bad executable
			isValid:           false,
			errStringContains: "image not available",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {

			var args []string

			args = append(args, "docker", "run")
			args = append(args, tc.imageName, "--", tc.executable)

			_, out, err := s.ExecuteTestCobraCommand(args...)
			s.Require().NoError(err, "Error submitting job")

			if !tc.isValid {
				s.Require().Contains(out, tc.errStringContains, "Error string does not contain expected string")
			} else {
				s.Require().NotContains(out, "Error", name+":"+"Error detected in output")
			}
		})
	}
}

func (s *DockerRunSuite) TestRun_InvalidImage() {
	// The error of Docker being unable to find the invalid image should get back to the user

	ctx := context.Background()

	_, out, err := s.ExecuteTestCobraCommand("docker", "run", "@", "--", "true")
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	s.T().Log(job)

	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{
		JobID:   job.ID,
		Include: "executions",
	})
	s.Require().NoError(err)
	s.T().Log(info)

	// NB(forrest): this test will break if we ever change out default retry policy.
	// Since the intention of the test is to assert that an execution failed with an
	// expected message we may consider adjusting the retry policy for this specific
	// test. Alternatively, we could reduce the complexity and assert the job
	// simply failed which is the expected behaviour for an invalid image
	s.Require().Len(info.Executions.Items, 2)
	s.Contains(info.Executions.Items[0].ComputeState.Message, `invalid image format: "@"`)
	s.Contains(info.Executions.Items[1].ComputeState.Message, `invalid image format: "@"`)
}

func (s *DockerRunSuite) TestRun_Timeout_DefaultValue() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"busybox:latest",
		"echo", "'hello world'",
	)
	s.NoError(err, "Error submitting job without defining a timeout value")

	j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

	s.Require().EqualValues(s.Config.JobDefaults.Batch.Task.Timeouts.TotalTimeout, j.Task().Timeouts.GetTotalTimeout(),
		"Did not fall back to default timeout value")
}

func (s *DockerRunSuite) TestRun_Timeout_DefinedValue() {
	const expectedTimeout = 999 * time.Second
	const expectedQueueTimeout = 888 * time.Second

	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"--timeout", fmt.Sprintf("%d", int64(expectedTimeout.Seconds())),
		"--queue-timeout", fmt.Sprintf("%d", int64(expectedQueueTimeout.Seconds())),
		"busybox:latest",
		"echo", "'hello world'",
	)
	s.NoError(err, "Error submitting job with a defined a timeout value")

	j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

	s.Require().Equal(expectedTimeout, j.Task().Timeouts.GetTotalTimeout())
	s.Require().Equal(expectedQueueTimeout, j.Task().Timeouts.GetQueueTimeout())
}

func (s *DockerRunSuite) TestRun_NoPublisher() {
	ctx := context.Background()

	_, out, err := s.ExecuteTestCobraCommand("docker", "run", "busybox:latest", "echo", "'hello world'")
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	s.T().Log(job)

	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{JobID: job.ID, Include: "executions"})
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Len(info.Executions.Items, 1)

	exec := info.Executions.Items[0]
	s.Require().Empty(exec.PublishedResult)

}

func (s *DockerRunSuite) TestRun_LocalPublisher() {
	ctx := context.Background()

	_, out, err := s.ExecuteTestCobraCommand("docker", "run", "-p", "local", "busybox:latest", "echo", "'hello world'")
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	s.T().Log(job)

	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{JobID: job.ID, Include: "executions"})
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Len(info.Executions.Items, 1)

	exec := info.Executions.Items[0]
	result := exec.PublishedResult
	s.Require().Equal(models.StorageSourceURL, result.Type)

	urlSpec, err := storage_url.DecodeSpec(result)
	s.Require().NoError(err)
	s.Require().Contains(urlSpec.URL, "http://127.0.0.1:", "URL does not contain expected prefix")
	s.Require().Contains(urlSpec.URL, fmt.Sprintf("%s.tar.gz", exec.ID), "URL does not contain expected file")

}
