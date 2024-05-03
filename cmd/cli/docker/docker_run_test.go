//go:build integration || !unit

package docker_test

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/google/uuid"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	ipfs_storage "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"

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

// TODO: #471 Refactor all of these tests to use common functionality; they're all very similar
func (s *DockerRunSuite) TestRun_GenericSubmit() {
	ctx := context.Background()
	randomUUID := uuid.New()
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"ubuntu",
		"echo",
		randomUUID.String(),
	)
	s.Require().NoError(err, "failed to submit job")

	testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
}

func (s *DockerRunSuite) TestRun_DryRun() {
	randomUUID := uuid.New()
	entrypointCommand := fmt.Sprintf("echo %s", randomUUID.String())
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"ubuntu",
		entrypointCommand,
		"--dry-run",
	)
	s.Require().NoError(err, "failed to submit job")

	s.Require().Contains(out, randomUUID.String(), "Dry run failed to contain UUID %s", randomUUID.String())

	var j *models.Job
	s.Require().NoError(model.YAMLUnmarshalWithMax([]byte(out), &j))
	s.Require().NotNil(j, "Failed to unmarshal job from dry run output")

	dockerSpec, err := models.DecodeSpecConfig[models.DockerEngineSpec](j.Task().Engine)
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
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
	}

	for i, tc := range tests {
		s.Run(fmt.Sprintf("numberOfJobs:%v", tc.numberOfJobs), func() {
			ctx := context.Background()

			swarmAddresses, err := s.Node.IPFSClient.SwarmAddresses(ctx)
			s.Require().NoError(err)

			_, out, err := s.ExecuteTestCobraCommand("docker", "run",
				"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
				"--wait",
				"--output-dir", s.T().TempDir(),
				"ubuntu",
				"--",
				"echo", "hello from docker submit wait",
			)
			s.Require().NoErrorf(err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
		})
	}
}

func (s *DockerRunSuite) TestRun_SubmitInputs() {
	s.T().Skip("TODO: test stack is not connected to public IPFS and can't resolve the CIDs")
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for i, tc := range tests {
		type (
			InputVolume struct {
				cid  string
				path string
				flag string
			}
		)

		testCids := []struct {
			inputVolumes []InputVolume
			err          error
		}{
			{inputVolumes: []InputVolume{{cid: "QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHA", path: "", flag: "-i"}}, err: nil},        // Fake CID, but well structured
			{inputVolumes: []InputVolume{{cid: "ipfs://QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHA", path: "", flag: "-i"}}, err: nil}, // Fake ipfs URI, but well structured
			{inputVolumes: []InputVolume{
				{cid: "QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHB", path: "", flag: "-i"},
				{cid: "QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHC", path: "", flag: "-i"}}, err: nil}, // 2x Fake CID, but well structured
			{inputVolumes: []InputVolume{
				{cid: "QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHD", path: "/CUSTOM_INPUT_PATH_1", flag: "-v"}}, err: nil}, // Fake CID, but well structured
			{inputVolumes: []InputVolume{
				{cid: "QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHE", path: "", flag: "-i"},
				{cid: "QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHF", path: "/CUSTOM_INPUT_PATH_2", flag: "-v"}}, err: nil}, // 2x Fake CID, but well structured

		}

		for _, tcids := range testCids {
			func() {
				ctx := context.Background()
				flagsArray := []string{"docker", "run"}
				for _, iv := range tcids.inputVolumes {
					ivString := iv.cid
					if iv.path != "" {
						ivString += fmt.Sprintf(":%s", iv.path)
					}
					flagsArray = append(flagsArray, iv.flag, ivString)
				}
				flagsArray = append(flagsArray, "ubuntu", "cat", "/inputs/foo.txt") // This doesn't exist, but shouldn't error

				_, out, err := s.ExecuteTestCobraCommand(flagsArray...)
				s.Require().NoError(err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

				s.Require().Equal(len(tcids.inputVolumes), len(j.Task().InputSources), "Number of job inputs != # of test inputs .")

				// Need to do the below because ordering is not guaranteed
				for _, tcidIV := range tcids.inputVolumes {
					testCIDinJobInputs := false
					for _, jobInput := range j.Task().InputSources {
						ipfsspec, err := models.DecodeSpecConfig[ipfs_storage.Source](jobInput.Source)
						s.Require().NoError(err)
						if tcidIV.cid == ipfsspec.CID {
							testCIDinJobInputs = true
							testPath := "/inputs"
							if tcidIV.path != "" {
								testPath = tcidIV.path
							}
							s.Require().Equal(testPath, jobInput.Target, "Test Path not equal to Path from job.")
							break
						}
					}
					s.Require().True(testCIDinJobInputs, "Test CID not in job inputs.")
				}
			}()
		}
	}
}

func (s *DockerRunSuite) TestRun_SubmitUrlInputs() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for i, tc := range tests {
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

		for _, turls := range testURLs {
			func() {
				ctx := context.Background()
				flagsArray := []string{"docker", "run"}

				flagsArray = append(flagsArray, turls.inputURL.flag, turls.inputURL.url)
				flagsArray = append(flagsArray, "ubuntu", "cat", fmt.Sprintf("%s/%s", turls.inputURL.pathInContainer, turls.inputURL.filename))

				_, out, err := s.ExecuteTestCobraCommand(flagsArray...)
				s.Require().NoError(err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

				s.Require().Equal(1, len(j.Task().InputSources), "Number of job urls != # of test urls.")
				urlspec, err := models.DecodeSpecConfig[urldownload.Source](j.Task().InputSources[0].Source)
				s.Require().NoError(err)
				s.Require().Equal(turls.inputURL.url, urlspec.URL, "Test URL not equal to URL from job.")
				s.Require().Equal(turls.inputURL.pathInContainer, j.Task().InputSources[0].Target, "Test Path not equal to Path from job.")

			}()
		}
	}
}

func (s *DockerRunSuite) TestRun_SubmitOutputs() {
	s.T().Skip("we no longer include outputs by default, re-enable this test when we do, or modify it.")
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for i, tc := range tests {
		type (
			OutputVolumes struct {
				name string
				path string
			}
		)

		testCids := []struct {
			outputVolumes []OutputVolumes
			correctLength int
			err           string
		}{
			{outputVolumes: []OutputVolumes{{name: "", path: ""}}, correctLength: 1, err: ""},                                                                     // Flag not provided
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME", path: "/outputs_1"}}, correctLength: 2, err: ""},                                                // Correct output flag
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME_2", path: "/outputs_2"}, {name: "OUTPUT_NAME_3", path: "/outputs_3"}}, correctLength: 3, err: ""}, // 2 correct output flags
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME_4", path: ""}}, correctLength: 0, err: "invalid output volume"},                                   // OV requested but no path (should error)
			{outputVolumes: []OutputVolumes{{name: "", path: "/outputs_4"}}, correctLength: 0, err: "invalid output volume"},                                      // OV requested but no name (should error)
		}

		for _, tcids := range testCids {
			func() {
				ctx := context.Background()
				flagsArray := []string{"docker", "run"}
				ovString := ""
				for _, ov := range tcids.outputVolumes {
					if ov.name != "" {
						ovString = ov.name
					}
					if ov.path != "" {
						ovString += fmt.Sprintf(":%s", ov.path)
					}
					if ovString != "" {
						flagsArray = append(flagsArray, "-o", ovString)
					}
				}
				flagsArray = append(flagsArray, "ubuntu", "echo", "'hello world'")

				_, out, err := s.ExecuteTestCobraCommand(flagsArray...)

				if tcids.err != "" {
					s.Require().Error(err)
					s.Require().Contains(string(out), "invalid output volume", "Missed detection of invalid output volume.")
					return // Go to next in loop
				}
				s.Require().NoError(err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

				s.Require().Equal(tcids.correctLength, len(j.Task().ResultPaths), "Number of job outputs != correct number.")

				// Need to do the below because ordering is not guaranteed
				for _, tcidOV := range tcids.outputVolumes {
					testNameinJobOutputs := false
					testPathinJobOutputs := false
					for _, jobOutput := range j.Task().ResultPaths {
						if tcidOV.name == "" {
							if jobOutput.Name == "outputs" {
								testNameinJobOutputs = true
							}
						} else {
							if tcidOV.name == jobOutput.Name {
								testNameinJobOutputs = true
							}
						}

						if tcidOV.path == "" {
							if jobOutput.Path == "/outputs" {
								testPathinJobOutputs = true
							}
						} else {
							if tcidOV.path == jobOutput.Path {
								testPathinJobOutputs = true
							}
						}
					}
					s.Require().True(testNameinJobOutputs, "Test OutputVolume Name not in job output names.")
					s.Require().True(testPathinJobOutputs, "Test OutputVolume Path not in job output paths.")
				}
			}()
		}
	}
}

func (s *DockerRunSuite) TestRun_CreatedAt() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			_, out, err := s.ExecuteTestCobraCommand("docker", "run",
				"ubuntu",
				"echo", "'hello world'",
			)
			s.NoError(err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

			s.Require().LessOrEqual(j.GetCreateTime(), time.Now(), "Created at time is not less than or equal to now.")

			oldStartTime, _ := time.Parse(time.RFC3339, "2021-01-01T01:01:01+00:00")
			s.Require().GreaterOrEqual(j.GetCreateTime(), oldStartTime, "Created at time is not greater or equal to 2022-01-01.")
		}()

	}
}

func (s *DockerRunSuite) TestRun_Annotations() {
	s.T().Skip("Labels have changed from a slice to a map, this test will not pass.")
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		// {numberOfJobs: 5}, // Test for five
	}

	annotationsToTest := []struct {
		Name          string
		Annotations   []string
		CorrectLength int
		BadCase       bool
	}{
		{Name: "1", Annotations: []string{""}, CorrectLength: 0, BadCase: false},                 // Label flag, no value, but correctly quoted
		{Name: "1.1", Annotations: []string{`""`}, CorrectLength: 0, BadCase: false},             // Label flag, no value, but correctly quoted
		{Name: "2", Annotations: []string{"a"}, CorrectLength: 1, BadCase: false},                // Annotations, string
		{Name: "3", Annotations: []string{"b", "1"}, CorrectLength: 2, BadCase: false},           // Annotations, string and int
		{Name: "4", Annotations: []string{`''`, `" "`}, CorrectLength: 0, BadCase: false},        // Annotations, some edge case characters
		{Name: "5", Annotations: []string{"🏳", "0", "🌈️"}, CorrectLength: 3, BadCase: false},     // Emojis
		{Name: "6", Annotations: []string{"ايطاليا"}, CorrectLength: 0, BadCase: false},          // Right to left
		{Name: "7", Annotations: []string{"\u202Btest\u202B"}, CorrectLength: 0, BadCase: false}, // Control charactel
		{Name: "8", Annotations: []string{"사회과학원", "어학연구소"}, CorrectLength: 0, BadCase: false},   // Two-byte characters
	}

	// allBadStrings := LoadBadStringsAnnotations()
	// for _, s := range allBadStrings {
	// 	strippedString := SafeStringStripper(s)
	// 	l := struct {
	// 		Annotations        []string
	// 		CorrectLength int
	// 		BadCase       bool
	// 	}{Annotations: []string{s}, CorrectLength: len(strippedString), BadCase: false}
	// 	AnnotationsToTest = append(AnnotationsToTest, l)
	// }

	for i, tc := range tests {
		func() {
			ctx := context.Background()

			for _, labelTest := range annotationsToTest {
				var args []string

				args = append(args, "docker", "run")
				for _, label := range labelTest.Annotations {
					args = append(args, "-l", label)
				}

				randNum, _ := crand.Int(crand.Reader, big.NewInt(10000))
				args = append(args, "ubuntu", "echo", fmt.Sprintf("'hello world - %s'", randNum.String()))

				_, out, err := s.ExecuteTestCobraCommand(args...)
				s.Require().NoError(err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

				if labelTest.BadCase {
					s.Require().Contains(out, "rror")
				} else {
					s.Require().NotNil(j, "Failed to get job with ID: %s", out)
					s.Require().NotContains(out, "rror", "'%s' caused an error", labelTest.Annotations)
					msg := fmt.Sprintf(`
Number of Annotations stored not equal to expected length.
Name: %s
Expected length: %d
Actual length: %d

Expected Annotations: %+v
Actual Annotations: %+v
`, labelTest.Name, len(labelTest.Annotations), len(j.Labels), labelTest.Annotations, j.Labels)
					s.Require().Equal(labelTest.CorrectLength, len(j.Labels), msg)
				}
			}
		}()
	}
}

func (s *DockerRunSuite) TestRun_EdgeCaseCLI() {
	tests := []struct {
		submitArgs []string
		fatalErr   bool
		errString  string
	}{
		{submitArgs: []string{"ubuntu", "-xoo -bar -baz"}, fatalErr: true, errString: "unknown shorthand flag"}, // submitting flag will fail if not separated with a --
		{submitArgs: []string{"ubuntu", "python -xoo -bar -baz"}, fatalErr: false, errString: ""},               // separating with -- should work and allow flags
		// {submitString: "-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images -o results:/output_images dpokidov/imagemagick -- magick mogrify -fx '((g-b)/(r+g+b))>0.02 ? 1 : 0' -resize 256x256 -quality 100 -path /output_images /input_images/*.jpg"},
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

			_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

			if tc.errString != "" {
				o := logBuf.String()
				s.Require().Contains(o, tc.errString, "Did not find expected error message in error string.\nExpected: %s\nActual: %s", tc.errString, o)
			}
		}()
	}
}

func (s *DockerRunSuite) TestRun_SubmitWorkdir() {
	tests := []struct {
		workdir   string
		errorCode int
	}{
		{workdir: "", errorCode: 0},
		{workdir: "/", errorCode: 0},
		{workdir: "./mydir", errorCode: 1},
		{workdir: "../mydir", errorCode: 1},
		{workdir: "http://foo.com", errorCode: 1},
		{workdir: "/foo//", errorCode: 0}, // double forward slash is allowed in unix
		{workdir: "/foo//bar", errorCode: 0},
	}

	for _, tc := range tests {
		s.T().Run(fmt.Sprintf("dir: %q, exitocode: %d", tc.workdir, tc.errorCode), func(t *testing.T) {
			ctx := context.Background()
			flagsArray := []string{"docker", "run"}
			flagsArray = append(flagsArray, "-w", tc.workdir)
			flagsArray = append(flagsArray, "ubuntu", "pwd")

			_, out, err := s.ExecuteTestCobraCommand(flagsArray...)

			if tc.errorCode != 0 {
				s.Require().NotNil(err, "Expected fatal error, but none found")
			} else {
				s.Require().NoError(err, "Error submitting job.")

				j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

				dockerSpec, err := models.DecodeSpecConfig[models.DockerEngineSpec](j.Task().Engine)
				s.Require().NoError(err)
				s.Require().Equal(tc.workdir, dockerSpec.WorkingDirectory, "Job workdir != test workdir.")
				s.Require().NoError(err, "Error in running command.")
			}
		})
	}
}

func (s *DockerRunSuite) TestRun_ExplodeVideos() {
	ctx := context.TODO()

	videos := []string{
		"Bird flying over the lake.mp4",
		"Calm waves on a rocky sea gulf.mp4",
		"Prominent Late Gothic styled architecture.mp4",
	}

	dirPath := s.T().TempDir()

	for _, video := range videos {
		err := os.WriteFile(
			filepath.Join(dirPath, video),
			[]byte(fmt.Sprintf("hello %s", video)),
			0644,
		)
		s.Require().NoError(err)
	}

	directoryCid, err := ipfs.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients([]*node.Node{s.Node})...)
	s.Require().NoError(err)

	allArgs := []string{
		"docker", "run",
		"--wait",
		"-i", fmt.Sprintf("ipfs://%s,dst=/inputs", directoryCid),
		"ubuntu", "echo", "hello",
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
				"ubuntu", "--", "perl", "-e", fmt.Sprintf(`print "=" x %d`, tc.inputLength),
			)
			s.Require().NoError(err, "Error submitting job. Name: %s. Expected Length: %s", name, tc.expectedLength)

			j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
			info, _, err := s.Client.Get(ctx, j.ID)
			s.Require().NoError(err)

			s.Len(info.State.Executions, 1)
			s.Len(info.State.Executions[0].RunOutput.STDOUT, int(tc.expectedLength.Bytes()))
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
		ctx := context.Background()
		var args []string

		args = append(args, "docker", "run")
		args = append(args, tc.inputFlags...)
		args = append(args, "ubuntu", "--", "ls", "/input")

		_, out, err := s.ExecuteTestCobraCommand(args...)
		s.Require().NoError(err, "Error submitting job")

		j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

		s.Require().Equal(tc.expectedVolumes, len(j.Task().InputSources))
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
			imageName:         "ubuntu", // Good image // TODO we consider an untagged image poor practice, fix this
			executable:        "ls",     // Good executable
			isValid:           true,
			errStringContains: "",
		},
		"bad-image-good-executable": {
			imageName:         "badimage", // Bad image
			executable:        "ls",       // Good executable
			isValid:           false,
			errStringContains: "Error submitting job",
		},
		"good-image-bad-executable": {
			imageName:         "ubuntu",        // Good image // TODO we consider an untagged image poor practice, fix this
			executable:        "BADEXECUTABLE", // Bad executable
			isValid:           false,
			errStringContains: "Error submitting job",
		},
		"bad-image-bad-executable": {
			imageName:         "badimage",      // Bad image
			executable:        "BADEXECUTABLE", // Bad executable
			isValid:           false,
			errStringContains: "Error submitting job",
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

	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{JobID: job.ID, Include: "executions"})
	s.Require().NoError(err)
	s.T().Log(info)
	s.Require().GreaterOrEqual(len(info.Executions.Executions), 1)

	s.Contains(info.Executions.Executions[0].ComputeState.Message, `Could not inspect image "@" - could be due to repo/image not existing`)
}

func (s *DockerRunSuite) TestRun_Timeout_DefaultValue() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"ubuntu",
		"echo", "'hello world'",
	)
	s.NoError(err, "Error submitting job without defining a timeout value")

	j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

	s.Require().Equal(node.TestRequesterConfig.JobDefaults.ExecutionTimeout, j.Task().Timeouts.ExecutionTimeout,
		"Did not fall back to default timeout value")
}

func (s *DockerRunSuite) TestRun_Timeout_DefinedValue() {
	const expectedTimeout = 999 * time.Second

	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("docker", "run",
		"--timeout", fmt.Sprintf("%d", int64(expectedTimeout.Seconds())),
		"ubuntu",
		"echo", "'hello world'",
	)
	s.NoError(err, "Error submitting job with a defined a timeout value")

	j := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

	s.Require().Equal(expectedTimeout, j.Task().Timeouts.ExecutionTimeout)
}

func (s *DockerRunSuite) TestRun_NoPublisher() {
	ctx := context.Background()

	_, out, err := s.ExecuteTestCobraCommand("docker", "run", "ubuntu", "echo", "'hello world'")
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	s.T().Log(job)

	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{JobID: job.ID, Include: "executions"})
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Len(info.Executions.Executions, 1)

	exec := info.Executions.Executions[0]
	s.Require().Empty(exec.PublishedResult)

}

func (s *DockerRunSuite) TestRun_LocalPublisher() {
	ctx := context.Background()

	_, out, err := s.ExecuteTestCobraCommand("docker", "run", "-p", "local", "ubuntu", "echo", "'hello world'")
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	s.T().Log(job)

	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{JobID: job.ID, Include: "executions"})
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Len(info.Executions.Executions, 1)

	exec := info.Executions.Executions[0]
	result, err := models.DecodeSpecConfig[urldownload.Source](exec.PublishedResult)
	s.Require().NoError(err)
	s.Require().Contains(result.URL, "http://127.0.0.1:", "URL does not contain expected prefix")
	s.Require().Contains(result.URL, fmt.Sprintf("%s.tgz", exec.ID), "URL does not contain expected file")

}
