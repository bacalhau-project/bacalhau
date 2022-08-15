package bacalhau

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/url"
	"os"
	"time"

	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/devstack"
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
	suite.Run(t, new(DockerRunSuite))
}

// Before all suite
func (suite *DockerRunSuite) SetupAllSuite() {
}

// Before each test
func (suite *DockerRunSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
	suite.rootCmd = RootCmd
}

func (suite *DockerRunSuite) TearDownTest() {
}

func (suite *DockerRunSuite) TearDownAllSuite() {

}

// TODO: Refactor all of these tests to use common functionality; they're all very similar

func (suite *DockerRunSuite) TestRun_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "docker", "run",
				"--api-host", host,
				"--api-port", port,
				"ubuntu echo 'hello world'",
			)
			require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			require.NoError(suite.T(), err)
			require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
}

func (suite *DockerRunSuite) TestRun_GPURequests() {
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
			log.Logger = log.With().Logger().Output(io.MultiWriter(Stdout, logBuf))

			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			allArgs := []string{"docker", "run", "--api-host", host, "--api-port", port}
			allArgs = append(allArgs, tc.submitArgs...)
			_, out, submitErr := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, allArgs...)

			if tc.fatalErr {
				require.Contains(suite.T(), out, tc.errString, "Did not find expected error message for fatalError in error string.\nExpected: %s\nActual: %s", tc.errString, out)
				return
			} else {
				require.NoError(suite.T(), submitErr, "Error submitting job. Run - Test-Number: %d - String: %s", i, tc.submitArgs)
			}

			require.True(suite.T(), !tc.fatalErr, "Expected fatal err, but submitted.")

			job, foundJob, getErr := c.Get(ctx, strings.TrimSpace(out))
			require.True(suite.T(), foundJob, "error getting job")
			require.NotNil(suite.T(), job, "Failed to get job with ID: %s\nErr: %+v", out, getErr)
			if tc.errString != "" {
				o := logBuf.String()
				require.Contains(suite.T(), o, tc.errString, "Did not find expected error message in error string.\nExpected: %s\nActual: %s", tc.errString, o)
			}
			require.Equal(suite.T(), tc.numGPUs, job.Spec.Resources.GPU, "Expected %d GPUs, but got %d", tc.numGPUs, job.Spec.Resources.GPU)
		}()
	}
}

func (suite *DockerRunSuite) TestRun_GenericSubmitWait() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
	}

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			devstack, cm := devstack.SetupTest(suite.T(), 1, 0, computenode.ComputeNodeConfig{})
			defer cm.Cleanup()

			dir, err := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitWait")
			require.NoError(suite.T(), err)

			swarmAddresses, err := devstack.Nodes[0].IpfsNode.SwarmAddresses()
			require.NoError(suite.T(), err)
			runDownloadFlags.IPFSSwarmAddrs = strings.Join(swarmAddresses, ",")
			runDownloadFlags.OutputDir = dir

			outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
			require.NoError(suite.T(), err)

			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "docker", "run",
				"--api-host", devstack.Nodes[0].APIServer.Host,
				"--api-port", fmt.Sprintf("%d", devstack.Nodes[0].APIServer.Port),
				"--wait",
				"--output-dir", outputDir,
				"ubuntu",
				"--",
				"echo", "hello from docker submit wait",
			)
			require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			client := publicapi.NewAPIClient(devstack.Nodes[0].APIServer.GetURI())
			job, _, err := client.Get(ctx, strings.TrimSpace(out))
			require.NoError(suite.T(), err)
			require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
}

func (suite *DockerRunSuite) TestRun_GenericSubmitLocal() {
	expectedStdout := "hello"
	args := []string{"docker", "run", "ubuntu", "echo", expectedStdout, "--local", "--wait", "--download"}
	done := capture()

	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocal-")
	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(suite.T(), err)
	}()
	runDownloadFlags.OutputDir = dir

	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
	out, _ := done()

	require.NoError(suite.T(), err)
	trimmedStdout := strings.TrimSpace(string(out))

	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

	runDownloadFlags.OutputDir = "."
}

func (suite *DockerRunSuite) TestRun_GenericSubmitLocalInput() {
	CID := "QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"
	args := []string{"docker", "run",
		"--local",
		"--wait",
		"--download",
		"-v", fmt.Sprintf("%s:/hello.txt", CID),
		"ubuntu",
		"cat", "hello.txt"}
	expectedStdout := "hello"

	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalInput-")
	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(suite.T(), err)
	}()
	runDownloadFlags.OutputDir = dir

	done := capture()
	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
	out, _ := done()

	require.NoError(suite.T(), err)
	trimmedStdout := strings.TrimSpace(string(out))
	fmt.Println(trimmedStdout)

	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

	runDownloadFlags.OutputDir = "."
}

func (suite *DockerRunSuite) TestRun_GenericSubmitLocalOutput() {
	args := []string{"docker", "run",
		"ubuntu",
		"--local",
		"--wait",
		"--download",
		"-w", "/outputs",
		"--",
		"/bin/bash", "-c", "printf hello > hello.txt"}
	expectedStdout := "hello"

	// done := capture()
	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
	if err != nil {
		fmt.Print(err)
	}
	// out, _ := done()

	require.NoError(suite.T(), err)
	content, _ := ioutil.ReadFile("volumes/outputs/hello.txt")
	out := string(content)
	trimmedStdout := strings.TrimSpace(string(out))
	fmt.Println(trimmedStdout)

	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

	runDownloadFlags.OutputDir = "."
}

func (suite *DockerRunSuite) TestRun_SubmitInputs() {
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
				c, cm := publicapi.SetupTests(suite.T())
				defer cm.Cleanup()

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

				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd,
					flagsArray...,
				)
				require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

				job, _, err := c.Get(ctx, strings.TrimSpace(out))
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)

				require.Equal(suite.T(), len(tcids.inputVolumes), len(job.Spec.Inputs), "Number of job inputs != # of test inputs .")

				// Need to do the below because ordering is not guaranteed
				for _, tcidIV := range tcids.inputVolumes {
					testCIDinJobInputs := false
					for _, jobInput := range job.Spec.Inputs {
						if tcidIV.cid == jobInput.Cid {
							testCIDinJobInputs = true
							testPath := "/inputs"
							if tcidIV.path != "" {
								testPath = tcidIV.path
							}
							require.Equal(suite.T(), testPath, jobInput.Path, "Test Path not equal to Path from job.")
							break
						}
					}
					require.True(suite.T(), testCIDinJobInputs, "Test CID not in job inputs.")
				}
			}()
		}
	}
}

func (suite *DockerRunSuite) TestRun_SubmitUrlInputs() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for i, tc := range tests {
		type (
			InputURL struct {
				url  string
				path string
				flag string
			}
		)

		testURLs := []struct {
			inputURLs []InputURL
			err       error
		}{
			{inputURLs: []InputURL{{url: "http://foo.com/bar.tar.gz", path: "/app/data.tar.gz", flag: "-u"}}, err: nil},
			{inputURLs: []InputURL{{url: "https://qaz.edu/sam.zip", path: "/app/sam.zip", flag: "-u"}}, err: nil},
			{inputURLs: []InputURL{{url: "https://ifps.io/CID", path: "/app/file.csv", flag: "-u"}}, err: nil},
		}
		jobOutputVolumes = []string{} // TODO reset all cli variables

		for _, turls := range testURLs {
			func() {
				ctx := context.Background()
				c, cm := publicapi.SetupTests(suite.T())
				defer cm.Cleanup()

				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				flagsArray := []string{"docker", "run",
					"--api-host", host,
					"--api-port", port}

				for _, iurl := range turls.inputURLs {
					iurlString := iurl.url
					if iurl.path != "" {
						iurlString += fmt.Sprintf(":%s", iurl.path)
					}
					flagsArray = append(flagsArray, iurl.flag, iurlString)
				}
				flagsArray = append(flagsArray, "ubuntu cat /app/foo_data.txt")

				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd,
					flagsArray...,
				)
				require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

				job, _, err := c.Get(ctx, strings.TrimSpace(out))
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)

				require.Equal(suite.T(), len(turls.inputURLs), len(job.Spec.Inputs), "Number of job urls != # of test urls.")

				// Need to do the below because ordering is not guaranteed
				for _, turlIU := range turls.inputURLs {
					testURLinJobInputs := false
					for _, jobInput := range job.Spec.Inputs {
						if turlIU.url == jobInput.URL {
							testURLinJobInputs = true
							testPath := "/app2"
							if turlIU.path != "" {
								testPath = turlIU.path
							}
							require.Equal(suite.T(), testPath, jobInput.Path, "Test Path not equal to Path from job.")
							break
						}
					}
					require.True(suite.T(), testURLinJobInputs, "Test URL not in job inputs.")
				}
			}()
		}
	}
}

func (suite *DockerRunSuite) TestRun_SubmitOutputs() {
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
				c, cm := publicapi.SetupTests(suite.T())
				defer cm.Cleanup()

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

				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd,
					flagsArray...,
				)
				if tcids.err != "" {
					require.Error(suite.T(), err, "Expected an error, but none provided. %+v", tcids)
					require.Contains(suite.T(), err.Error(), "invalid output volume", "Missed detection of invalid output volume.")
					return // Go to next in loop
				}
				require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				job, _, err := c.Get(ctx, strings.TrimSpace(out))
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)

				require.Equal(suite.T(), tcids.correctLength, len(job.Spec.Outputs), "Number of job outputs != correct number.")

				// Need to do the below because ordering is not guaranteed
				for _, tcidOV := range tcids.outputVolumes {
					testNameinJobOutputs := false
					testPathinJobOutputs := false
					for _, jobOutput := range job.Spec.Outputs {
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
					require.True(suite.T(), testNameinJobOutputs, "Test OutputVolume Name not in job output names.")
					require.True(suite.T(), testPathinJobOutputs, "Test OutputVolume Path not in job output paths.")
				}
			}()
		}
	}
}

func (suite *DockerRunSuite) TestRun_CreatedAt() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "docker", "run",
				"--api-host", host,
				"--api-port", port,
				"ubuntu echo 'hello world'",
			)
			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			require.NoError(suite.T(), err)
			require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
			require.LessOrEqual(suite.T(), job.CreatedAt, time.Now(), "Created at time is not less than or equal to now.")

			oldStartTime, _ := time.Parse(time.RFC3339, "2021-01-01T01:01:01+00:00")
			require.GreaterOrEqual(suite.T(), job.CreatedAt, oldStartTime, "Created at time is not greater or equal to 2022-01-01.")
		}()

	}
}

func (suite *DockerRunSuite) TestRun_Annotations() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		// {numberOfJobs: 5}, // Test for five
	}

	var logBuf = new(bytes.Buffer)
	var Stdout = struct{ io.Writer }{os.Stdout}
	log.Logger = log.With().Logger().Output(io.MultiWriter(Stdout, logBuf))

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
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			for _, labelTest := range annotationsToTest {
				// log.Warn().Msgf("%s - Args: %+v", labelTest.Name, os.Args)
				parsedBasedURI, err := url.Parse(c.BaseURI)
				require.NoError(suite.T(), err)

				host, port, err := net.SplitHostPort(parsedBasedURI.Host)
				require.NoError(suite.T(), err)

				var args []string

				args = append(args, "docker", "run", "--api-host", host, "--api-port", port)
				for _, label := range labelTest.Annotations {
					args = append(args, "-l", label)
				}

				randNum, _ := crand.Int(crand.Reader, big.NewInt(10000))
				args = append(args, fmt.Sprintf("ubuntu echo 'hello world - %s'", randNum.String()))

				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
				require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				testJob, _, err := c.Get(ctx, strings.TrimSpace(out))
				require.NoError(suite.T(), err)

				if labelTest.BadCase {
					require.Contains(suite.T(), out, "rror")
				} else {
					require.NotNil(suite.T(), testJob, "Failed to get job with ID: %s", out)
					require.NotContains(suite.T(), out, "rror", "'%s' caused an error", labelTest.Annotations)
					msg := fmt.Sprintf(`
Number of Annotations stored not equal to expected length.
Name: %s
Expected length: %d
Actual length: %d

Expected Annotations: %+v
Actual Annotations: %+v
`, labelTest.Name, len(labelTest.Annotations), len(testJob.Spec.Annotations), labelTest.Annotations, testJob.Spec.Annotations)
					require.Equal(suite.T(), labelTest.CorrectLength, len(testJob.Spec.Annotations), msg)
				}
			}
		}()
	}
}

func (suite *DockerRunSuite) TestRun_EdgeCaseCLI() {
	tests := []struct {
		submitArgs []string
		fatalErr   bool
		errString  string
	}{
		{submitArgs: []string{"ubuntu", "-foo -bar -baz"}, fatalErr: true, errString: "unknown shorthand flag"},     // submitting flag will fail if not separated with a --
		{submitArgs: []string{"ubuntu", "python -foo -bar -baz"}, fatalErr: false, errString: ""},                   // separating with -- should work and allow flags
		{submitArgs: []string{"ubuntu", "baz -foo -bar -baz *.jpg"}, fatalErr: false, errString: "contains a glob"}, // contains a glob, and should fail
		{submitArgs: []string{"ubuntu", "/bin/bash *.jpg"}, fatalErr: false, errString: ""},                         // contains a glob but starts with a shell (and a space)
		// {submitString: "-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images -o results:/output_images dpokidov/imagemagick -- magick mogrify -fx '((g-b)/(r+g+b))>0.02 ? 1 : 0' -resize 256x256 -quality 100 -path /output_images /input_images/*.jpg"},
	}

	for i, tc := range tests {
		func() {

			var logBuf = new(bytes.Buffer)
			var Stdout = struct{ io.Writer }{os.Stdout}
			log.Logger = log.With().Logger().Output(io.MultiWriter(Stdout, logBuf))

			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			allArgs := []string{"docker", "run", "--api-host", host, "--api-port", port}
			allArgs = append(allArgs, tc.submitArgs...)
			_, out, submitErr := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, allArgs...)

			if tc.fatalErr {
				require.Contains(suite.T(), out, tc.errString, "Did not find expected error message for fatalError in error string.\nExpected: %s\nActual: %s", tc.errString, out)
				return
			} else {
				require.NoError(suite.T(), submitErr, "Error submitting job. Run - Test-Number: %d - String: %s", i, tc.submitArgs)
			}

			require.True(suite.T(), !tc.fatalErr, "Expected fatal err, but submitted.")

			job, foundJob, getErr := c.Get(ctx, strings.TrimSpace(out))
			require.True(suite.T(), foundJob, "error getting job")
			require.NotNil(suite.T(), job, "Failed to get job with ID: %s\nErr: %+v", out, getErr)
			if tc.errString != "" {
				o := logBuf.String()
				require.Contains(suite.T(), o, tc.errString, "Did not find expected error message in error string.\nExpected: %s\nActual: %s", tc.errString, o)
			}
		}()
	}
}

func (suite *DockerRunSuite) TestRun_SubmitWorkdir() {
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

	// TODO reset all cli variables
	jobOutputVolumes = []string{}

	for _, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			flagsArray := []string{"docker", "run",
				"--api-host", host,
				"--api-port", port}
			flagsArray = append(flagsArray, "-w", tc.workdir)
			flagsArray = append(flagsArray, "ubuntu pwd")

			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd,
				flagsArray...,
			)

			if tc.error_code != 0 {
				require.Error(suite.T(), err)
			} else {
				require.NoError(suite.T(), err, "Error submitting job.")
				job, _, err := c.Get(ctx, strings.TrimSpace(out))
				require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
				require.Equal(suite.T(), tc.workdir, job.Spec.Docker.WorkingDir, "Job workdir != test workdir.")
				require.NoError(suite.T(), err, "Error in running command.")
			}
		}()
	}
}

func (suite *DockerRunSuite) TestRun_ExplodeVideos() {
	const nodeCount = 1

	videos := []string{
		"Bird flying over the lake.mp4",
		"Calm waves on a rocky sea gulf.mp4",
		"Prominent Late Gothic styled architecture.mp4",
	}

	stack, cm := devstack.SetupTest(
		suite.T(),
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer cm.Cleanup()

	dirPath, err := os.MkdirTemp("", "sharding-test")
	require.NoError(suite.T(), err)
	for _, video := range videos {
		err = os.WriteFile(
			fmt.Sprintf("%s/%s", dirPath, video),
			[]byte(fmt.Sprintf("hello %s", video)),
			0644,
		)
		require.NoError(suite.T(), err)
	}

	directoryCid, err := stack.AddFileToNodes(nodeCount, dirPath)
	require.NoError(suite.T(), err)

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

	_, _, submitErr := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, allArgs...)
	require.NoError(suite.T(), submitErr)
}
