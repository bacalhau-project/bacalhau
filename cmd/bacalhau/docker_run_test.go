//go:build !(unit && (windows || darwin))

package bacalhau

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"sigs.k8s.io/yaml"

	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	devstack_tests "github.com/filecoin-project/bacalhau/pkg/test/devstack"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DockerRunSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDockerRunSuite(t *testing.T) {
	Fatal = FakeFatalErrorHandler
	suite.Run(t, new(DockerRunSuite))
}

// Before all suite
func (s *DockerRunSuite) SetupSuite() {
}

// Before each test
func (s *DockerRunSuite) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting())
	s.rootCmd = RootCmd
}

func (s *DockerRunSuite) TearDownTest() {
}

func (s *DockerRunSuite) TearDownSuite() {

}

// TODO: #471 Refactor all of these tests to use common functionality; they're all very similar
func (s *DockerRunSuite) TestRun_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			*ODR = *NewDockerRunOptions()

			randomUUID := uuid.New()
			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
				"--api-host", host,
				"--api-port", port,
				fmt.Sprintf("ubuntu echo %s", randomUUID.String()),
			)
			require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			_ = testutils.GetJobFromTestOutput(ctx, s.T(), c, out)
		}()
	}
}

func (s *DockerRunSuite) TestRun_DryRun() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
	}

	for i, tc := range tests {
		func() {
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			*ODR = *NewDockerRunOptions()

			randomUUID := uuid.New()
			entrypointCommand := fmt.Sprintf("echo %s", randomUUID.String())

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
				"--api-host", host,
				"--api-port", port,
				"ubuntu",
				entrypointCommand,
				"--dry-run",
			)
			require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			require.NoError(s.T(), err)
			require.Contains(s.T(), string(out), randomUUID.String(), "Dry run failed to contain UUID %s", randomUUID.String())

			var j *model.Job
			yaml.Unmarshal([]byte(out), &j)
			require.NotNil(s.T(), j, "Failed to unmarshal job from dry run output")
			require.Equal(s.T(), j.Spec.Docker.Entrypoint[0], entrypointCommand, "Dry run job should not have an ID")
		}()
	}
}

func (s *DockerRunSuite) TestRun_GPURequests() {
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
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			*ODR = *NewDockerRunOptions()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			allArgs := []string{"docker", "run", "--api-host", host, "--api-port", port}
			allArgs = append(allArgs, tc.submitArgs...)
			_, out, submitErr := ExecuteTestCobraCommand(s.T(), s.rootCmd, allArgs...)

			if tc.fatalErr {
				require.Contains(s.T(), out, tc.errString, "Did not find expected error message for fatalError in error string.\nExpected: %s\nActual: %s", tc.errString, out)
				return
			} else {
				require.NoError(s.T(), submitErr, "Error submitting job. Run - Test-Number: %d - String: %s", i, tc.submitArgs)
			}

			require.True(s.T(), !tc.fatalErr, "Expected fatal err, but submitted.")

			j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

			if tc.errString != "" {
				o := logBuf.String()
				require.Contains(s.T(), o, tc.errString, "Did not find expected error message in error string.\nExpected: %s\nActual: %s", tc.errString, o)
			}
			require.Equal(s.T(), tc.numGPUs, j.Spec.Resources.GPU, "Expected %d GPUs, but got %d", tc.numGPUs, j.Spec.Resources.GPU)
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
			devstack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false, computenode.ComputeNodeConfig{})

			*ODR = *NewDockerRunOptions()

			dir := s.T().TempDir()

			swarmAddresses, err := devstack.Nodes[0].IPFSClient.SwarmAddresses(ctx)
			require.NoError(s.T(), err)
			ODR.DownloadFlags.IPFSSwarmAddrs = strings.Join(swarmAddresses, ",")
			ODR.DownloadFlags.OutputDir = dir

			outputDir := s.T().TempDir()

			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
				"--api-host", devstack.Nodes[0].APIServer.Host,
				"--api-port", fmt.Sprintf("%d", devstack.Nodes[0].APIServer.Port),
				"--wait",
				"--output-dir", outputDir,
				"ubuntu",
				"--",
				"echo", "hello from docker submit wait",
			)
			require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			c := publicapi.NewAPIClient(fmt.Sprintf("http://%s:%d", devstack.Nodes[0].APIServer.Host, devstack.Nodes[0].APIServer.Port))
			_ = testutils.GetJobFromTestOutput(ctx, s.T(), c, out)
		})
	}
}

func (s *DockerRunSuite) TestRun_SubmitInputs() {
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
			{inputVolumes: []InputVolume{{cid: "QmZUCdf9ZdpbHdr9pU8XjdUMKutKa1aVSrLZZWC4uY4pHA", path: "", flag: "-i"}}, err: nil}, // Fake CID, but well structured
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
				c, cm := publicapi.SetupRequesterNodeForTests(s.T())
				defer cm.Cleanup()

				*ODR = *NewDockerRunOptions()

				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				flagsArray := []string{"docker", "run",
					"--api-host", host,
					"--api-port", port}
				for _, iv := range tcids.inputVolumes {
					ivString := iv.cid
					if iv.path != "" {
						ivString += fmt.Sprintf(":%s", iv.path)
					}
					flagsArray = append(flagsArray, iv.flag, ivString)
				}
				flagsArray = append(flagsArray, "ubuntu cat /inputs/foo.txt") // This doesn't exist, but shouldn't error

				_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd,
					flagsArray...,
				)
				require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

				require.Equal(s.T(), len(tcids.inputVolumes), len(j.Spec.Inputs), "Number of job inputs != # of test inputs .")

				// Need to do the below because ordering is not guaranteed
				for _, tcidIV := range tcids.inputVolumes {
					testCIDinJobInputs := false
					for _, jobInput := range j.Spec.Inputs {
						if tcidIV.cid == jobInput.CID {
							testCIDinJobInputs = true
							testPath := "/inputs"
							if tcidIV.path != "" {
								testPath = tcidIV.path
							}
							require.Equal(s.T(), testPath, jobInput.Path, "Test Path not equal to Path from job.")
							break
						}
					}
					require.True(s.T(), testCIDinJobInputs, "Test CID not in job inputs.")
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

	Fatal = FakeFatalErrorHandler

	for i, tc := range tests {
		type (
			InputURL struct {
				url             string
				pathInContainer string
				flag            string
			}
		)

		// For URLs, the input should be a file, the output a directory
		// Internally the URL storage provider appends the filename to the directory path
		testURLs := []struct {
			inputURLs []InputURL
			err       error
		}{
			{inputURLs: []InputURL{{url: "http://foo.com/bar.tar.gz", pathInContainer: "/inputs", flag: "-u"}}, err: nil},
			{inputURLs: []InputURL{{url: "https://qaz.edu/sam.zip", pathInContainer: "/inputs", flag: "-u"}}, err: nil},
			{inputURLs: []InputURL{{url: "https://ifps.io/CID", pathInContainer: "/inputs", flag: "-u"}}, err: nil},
		}

		for _, turls := range testURLs {
			func() {
				ctx := context.Background()
				c, cm := publicapi.SetupRequesterNodeForTests(s.T())
				defer cm.Cleanup()

				*ODR = *NewDockerRunOptions()

				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				flagsArray := []string{"docker", "run",
					"--api-host", host,
					"--api-port", port}

				for _, iurl := range turls.inputURLs {
					iurlString := iurl.url
					flagsArray = append(flagsArray, iurl.flag, iurlString)
				}
				flagsArray = append(flagsArray, "ubuntu cat /app/foo_data.txt")

				_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd,
					flagsArray...,
				)
				require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

				require.Equal(s.T(), len(turls.inputURLs), len(j.Spec.Inputs), "Number of job urls != # of test urls.")

				// Need to do the below because ordering is not guaranteed
				for _, turlIU := range turls.inputURLs {
					testURLinJobInputs := false
					for _, jobInput := range j.Spec.Inputs {
						if turlIU.url == jobInput.URL && turlIU.pathInContainer == jobInput.Path {
							testURLinJobInputs = true
						}
					}
					require.True(s.T(), testURLinJobInputs, "Test URL not in job inputs: %s", turlIU.url)

				}
			}()
		}
	}
}

func (s *DockerRunSuite) TestRun_SubmitOutputs() {
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
				Fatal = FakeFatalErrorHandler

				ctx := context.Background()
				c, cm := publicapi.SetupRequesterNodeForTests(s.T())
				defer cm.Cleanup()

				*ODR = *NewDockerRunOptions()

				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				flagsArray := []string{"docker", "run",
					"--api-host", host,
					"--api-port", port}
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
				flagsArray = append(flagsArray, "ubuntu echo 'hello world'")

				_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd,
					flagsArray...,
				)

				if tcids.err != "" {
					firstFatalError, err := testutils.FirstFatalError(s.T(), out)

					require.NoError(s.T(), err, "Error unmarshaling errors. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)
					require.Greater(s.T(), firstFatalError.Code, 0, "Expected an error, but none provided. %+v", tcids)
					require.Contains(s.T(), firstFatalError.Message, "invalid output volume", "Missed detection of invalid output volume.")
					return // Go to next in loop
				}
				require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

				require.Equal(s.T(), tcids.correctLength, len(j.Spec.Outputs), "Number of job outputs != correct number.")

				// Need to do the below because ordering is not guaranteed
				for _, tcidOV := range tcids.outputVolumes {
					testNameinJobOutputs := false
					testPathinJobOutputs := false
					for _, jobOutput := range j.Spec.Outputs {
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
					require.True(s.T(), testNameinJobOutputs, "Test OutputVolume Name not in job output names.")
					require.True(s.T(), testPathinJobOutputs, "Test OutputVolume Path not in job output paths.")
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
			*ODR = *NewDockerRunOptions()

			ctx := context.Background()
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
				"--api-host", host,
				"--api-port", port,
				"ubuntu",
				"echo 'hello world'",
			)
			assert.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

			require.LessOrEqual(s.T(), j.CreatedAt, time.Now(), "Created at time is not less than or equal to now.")

			oldStartTime, _ := time.Parse(time.RFC3339, "2021-01-01T01:01:01+00:00")
			require.GreaterOrEqual(s.T(), j.CreatedAt, oldStartTime, "Created at time is not greater or equal to 2022-01-01.")
		}()

	}
}

func (s *DockerRunSuite) TestRun_Annotations() {
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
		{Name: "1", Annotations: []string{""}, CorrectLength: 0, BadCase: false},               // Label flag, no value, but correctly quoted
		{Name: "1.1", Annotations: []string{`""`}, CorrectLength: 0, BadCase: false},           // Label flag, no value, but correctly quoted
		{Name: "2", Annotations: []string{"a"}, CorrectLength: 1, BadCase: false},              // Annotations, string
		{Name: "3", Annotations: []string{"b", "1"}, CorrectLength: 2, BadCase: false},         // Annotations, string and int
		{Name: "4", Annotations: []string{`''`, `" "`}, CorrectLength: 0, BadCase: false},      // Annotations, some edge case characters
		{Name: "5", Annotations: []string{"🏳", "0", "🌈️"}, CorrectLength: 3, BadCase: false},   // Emojis
		{Name: "6", Annotations: []string{"ايطاليا"}, CorrectLength: 0, BadCase: false},        // Right to left
		{Name: "7", Annotations: []string{"‫test‫"}, CorrectLength: 0, BadCase: false},         // Control charactel
		{Name: "8", Annotations: []string{"사회과학원", "어학연구소"}, CorrectLength: 0, BadCase: false}, // Two-byte characters
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
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			for _, labelTest := range annotationsToTest {
				*ODR = *NewDockerRunOptions()

				// log.Warn().Msgf("%s - Args: %+v", labelTest.Name, os.Args)
				parsedBasedURI, err := url.Parse(c.BaseURI)
				require.NoError(s.T(), err)

				host, port, err := net.SplitHostPort(parsedBasedURI.Host)
				require.NoError(s.T(), err)

				var args []string

				args = append(args, "docker", "run", "--api-host", host, "--api-port", port)
				for _, label := range labelTest.Annotations {
					args = append(args, "-l", label)
				}

				randNum, _ := crand.Int(crand.Reader, big.NewInt(10000))
				args = append(args, fmt.Sprintf("ubuntu echo 'hello world - %s'", randNum.String()))

				_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, args...)
				require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

				if labelTest.BadCase {
					require.Contains(s.T(), out, "rror")
				} else {
					require.NotNil(s.T(), j, "Failed to get job with ID: %s", out)
					require.NotContains(s.T(), out, "rror", "'%s' caused an error", labelTest.Annotations)
					msg := fmt.Sprintf(`
Number of Annotations stored not equal to expected length.
Name: %s
Expected length: %d
Actual length: %d

Expected Annotations: %+v
Actual Annotations: %+v
`, labelTest.Name, len(labelTest.Annotations), len(j.Spec.Annotations), labelTest.Annotations, j.Spec.Annotations)
					require.Equal(s.T(), labelTest.CorrectLength, len(j.Spec.Annotations), msg)
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
		{submitArgs: []string{"ubuntu", "-foo -bar -baz"}, fatalErr: true, errString: "unknown shorthand flag"}, // submitting flag will fail if not separated with a --
		{submitArgs: []string{"ubuntu", "python -foo -bar -baz"}, fatalErr: false, errString: ""},               // separating with -- should work and allow flags
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
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			*ODR = *NewDockerRunOptions()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			allArgs := []string{"docker", "run", "--api-host", host, "--api-port", port}
			allArgs = append(allArgs, tc.submitArgs...)
			_, out, submitErr := ExecuteTestCobraCommand(s.T(), s.rootCmd, allArgs...)

			if tc.fatalErr {
				require.Contains(s.T(), out, tc.errString, "Did not find expected error message for fatalError in error string.\nExpected: %s\nActual: %s", tc.errString, out)
				return
			} else {
				require.NoError(s.T(), submitErr, "Error submitting job. Run - Test-Number: %d - String: %s", i, tc.submitArgs)
			}

			require.True(s.T(), !tc.fatalErr, "Expected fatal err, but submitted.")

			_ = testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

			if tc.errString != "" {
				o := logBuf.String()
				require.Contains(s.T(), o, tc.errString, "Did not find expected error message in error string.\nExpected: %s\nActual: %s", tc.errString, o)
			}
		}()
	}
}

func (s *DockerRunSuite) TestRun_SubmitWorkdir() {
	tests := []struct {
		workdir    string
		error_code int
	}{
		{workdir: "", error_code: 0},
		{workdir: "/", error_code: 0},
		{workdir: "./mydir", error_code: 1},
		{workdir: "../mydir", error_code: 1},
		{workdir: "http://foo.com", error_code: 1},
		{workdir: "/foo//", error_code: 0}, // double forward slash is allowed in unix
		{workdir: "/foo//bar", error_code: 0},
	}

	for _, tc := range tests {
		func() {
			Fatal = FakeFatalErrorHandler

			ctx := context.Background()
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			*ODR = *NewDockerRunOptions()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			flagsArray := []string{"docker", "run",
				"--api-host", host,
				"--api-port", port}
			flagsArray = append(flagsArray, "-w", tc.workdir)
			flagsArray = append(flagsArray, "ubuntu pwd")

			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd,
				flagsArray...,
			)

			if tc.error_code != 0 {
				fatalError, err := testutils.FirstFatalError(s.T(), out)
				require.NoError(s.T(), err, "Error getting first fatal error")

				require.NotNil(s.T(), fatalError, "Expected fatal error, but none found")
			} else {
				require.NoError(s.T(), err, "Error submitting job.")

				j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

				require.Equal(s.T(), tc.workdir, j.Spec.Docker.WorkingDirectory, "Job workdir != test workdir.")
				require.NoError(s.T(), err, "Error in running command.")
			}
		}()
	}
}

func (s *DockerRunSuite) TestRun_ExplodeVideos() {
	ctx := context.TODO()
	const nodeCount = 1

	videos := []string{
		"Bird flying over the lake.mp4",
		"Calm waves on a rocky sea gulf.mp4",
		"Prominent Late Gothic styled architecture.mp4",
	}

	stack, _ := devstack_tests.SetupTest(
		ctx,
		s.T(),
		nodeCount,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
	)

	*ODR = *NewDockerRunOptions()

	dirPath := s.T().TempDir()

	for _, video := range videos {
		err := os.WriteFile(
			filepath.Join(dirPath, video),
			[]byte(fmt.Sprintf("hello %s", video)),
			0644,
		)
		require.NoError(s.T(), err)
	}

	directoryCid, err := devstack.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(s.T(), err)

	parsedBasedURI, _ := url.Parse(stack.Nodes[0].APIServer.GetURI())
	host, port, _ := net.SplitHostPort(parsedBasedURI.Host)

	allArgs := []string{
		"docker", "run",
		"--api-host", host,
		"--api-port", port,
		"--wait",
		"-v", fmt.Sprintf("%s:/inputs", directoryCid),
		"--sharding-base-path", "/inputs",
		"--sharding-glob-pattern", "*.mp4",
		"--sharding-batch-size", "1",
		"ubuntu", "echo", "hello",
	}

	_, _, submitErr := ExecuteTestCobraCommand(s.T(), s.rootCmd, allArgs...)
	require.NoError(s.T(), submitErr)
}

func (s *DockerRunSuite) TestRun_Deterministic_Verifier() {
	ctx := context.Background()

	apiSubmitJob := func(
		apiClient *publicapi.APIClient,
		args devstack_tests.DeterministicVerifierTestArgs,
	) (string, error) {

		parsedBasedURI, _ := url.Parse(apiClient.BaseURI)
		host, port, _ := net.SplitHostPort(parsedBasedURI.Host)

		ODR.Inputs = make([]string, 0)
		ODR.InputVolumes = make([]string, 0)

		_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd,
			"docker", "run",
			"--api-host", host,
			"--api-port", port,
			"-v", "123:/",
			"--verifier", "deterministic",
			"--concurrency", strconv.Itoa(args.NodeCount),
			"--confidence", strconv.Itoa(args.Confidence),
			"--sharding-glob-pattern", "/data/*.txt",
			"--sharding-batch-size", "1",
			"ubuntu", "echo", "hello",
		)

		if err != nil {
			return "", err
		}
		j := testutils.GetJobFromTestOutput(ctx, s.T(), apiClient, out)
		return j.ID, nil
	}

	devstack_tests.RunDeterministicVerifierTests(ctx, s.T(), apiSubmitJob)
}

func (s *DockerRunSuite) TestTruncateReturn() {
	system.MaxStderrReturnLengthInBytes = 10 // Make it artificially small for this run

	tests := map[string]struct {
		inputLength    int
		expectedLength int
		truncated      bool
	}{
		// "zero length": {inputLength: 0, truncated: false, expectedLength: 0},
		// "one length":  {inputLength: 1, truncated: false, expectedLength: 1},
		"maxLength - 1": {inputLength: system.MaxStdoutReturnLengthInBytes - 1,
			truncated:      false,
			expectedLength: system.MaxStdoutReturnLengthInBytes - 1},
		"maxLength": {inputLength: system.MaxStdoutReturnLengthInBytes,
			truncated:      false,
			expectedLength: system.MaxStdoutReturnLengthInBytes},
		"maxLength + 1": {inputLength: system.MaxStdoutReturnLengthInBytes + 1,
			truncated:      true,
			expectedLength: system.MaxStdoutReturnLengthInBytes},
		"maxLength + 10000": {inputLength: system.MaxStdoutReturnLengthInBytes * 10,
			truncated: true, expectedLength: system.MaxStdoutReturnLengthInBytes},
	}

	for name, tc := range tests {
		s.T().Run(name, func(t *testing.T) {
			ctx := context.Background()
			c, cm := publicapi.SetupRequesterNodeForTests(s.T())
			defer cm.Cleanup()

			*ODR = *NewDockerRunOptions()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			flagsArray := []string{"docker", "run",
				"--api-host", host,
				"--api-port", port}

			flagsArray = append(flagsArray, fmt.Sprintf(`ubuntu perl -e "print \"=\" x %d"`, tc.inputLength))

			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd,
				flagsArray...,
			)
			require.NoError(s.T(), err, "Error submitting job. Name: %s. Expected Length: %s", name, tc.expectedLength)

			_ = testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

			// require.Equal(suite.T(), len(turls.inputURLs), len(job.Spec.Inputs), "Number of job urls != # of test urls.")

		})
	}
}

func (s *DockerRunSuite) TestRun_MutlipleURLs() {

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
			[]string{"-u", "http://127.0.0.1:/inputs/url1.txt"},
		},
		{
			2,
			[]string{
				"-u", "http://127.0.0.1:/inputs/url1.txt",
				"-u", "http://127.0.0.1:/inputs/url2.txt",
			},
		},
		{
			2,
			[]string{
				"-u", "http://127.0.0.1:/inputs/url1.txt,http://127.0.0.1:/inputs/url2.txt",
			},
		},
	}

	for _, tc := range tests {
		ctx := context.Background()
		c, cm := publicapi.SetupRequesterNodeForTests(s.T())
		defer cm.Cleanup()

		*ODR = *NewDockerRunOptions()

		parsedBasedURI, _ := url.Parse(c.BaseURI)
		host, port, _ := net.SplitHostPort(parsedBasedURI.Host)

		args := []string{}

		args = append(args, "docker", "run",
			"--api-host", host,
			"--api-port", port,
		)
		args = append(args, tc.inputFlags...)
		args = append(args, "ubuntu", "--", "ls", "/input")

		_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, args...)
		require.NoError(s.T(), err, "Error submitting job")

		j := testutils.GetJobFromTestOutput(ctx, s.T(), c, out)

		require.Equal(s.T(), tc.expectedVolumes, len(j.Spec.Inputs))
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
			imageName:         "ubuntu", // Good image
			executable:        "ls",     // Good executable
			isValid:           true,
			errStringContains: "",
		},
		"bad-image-good-executable": {
			imageName:         "badimage", // Bad image
			executable:        "ls",       // Good executable
			isValid:           false,
			errStringContains: "Could not pull image",
		},
		"good-image-bad-executable": {
			imageName:         "ubuntu",        // Good image
			executable:        "BADEXECUTABLE", // Bad executable
			isValid:           false,
			errStringContains: "Executable file not found",
		},
		"bad-image-bad-executable": {
			imageName:         "badimage",      // Bad image
			executable:        "BADEXECUTABLE", // Bad executable
			isValid:           false,
			errStringContains: "Could not pull image",
		},
	}

	ctx := context.TODO()
	stack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false, computenode.ComputeNodeConfig{})

	for name, tc := range tests {
		*ODR = *NewDockerRunOptions()

		parsedBasedURI, _ := url.Parse(stack.Nodes[0].APIServer.GetURI())
		host, port, _ := net.SplitHostPort(parsedBasedURI.Host)

		args := []string{}

		args = append(args, "docker", "run",
			"--api-host", host,
			"--api-port", port,
		)
		args = append(args, tc.imageName, "--", tc.executable)

		_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, args...)
		require.NoError(s.T(), err, "Error submitting job")

		if !tc.isValid {
			require.Contains(s.T(), out, tc.errStringContains, "Error string does not contain expected string")
		} else {
			require.NotContains(s.T(), out, "Error", name+":"+"Error detected in output")
		}
	}
}
