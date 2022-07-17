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
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
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
			require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			require.NoError(suite.T(), err)
			require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
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
			err           error
		}{
			{outputVolumes: []OutputVolumes{{name: "", path: ""}}, err: nil}, // Flag not provided
		}

		_ = i
		_ = tc
		_ = testCids

		// for _, tcids := range testCids {
		// 	func() {
		// 		ctx := context.Background()
		// 		c, cm := publicapi.SetupTests(suite.T())
		// 		defer cm.Cleanup()

		// parsedBasedURI, _ := url.Parse(c.BaseURI)
		// host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
		// flagsArray := []string{"docker", "run",
		// 	"--api-host", host,
		// 	"--api-port", port}
		// for _, ov := range tcids.outputVolumes {
		// 	ovString := iv.cid
		// 	if iv.path != "" {
		// 		ivString += fmt.Sprintf(":%s", iv.path)
		// 	}
		// 	flagsArray = append(flagsArray, iv.flag, ivString)
		// }
		// flagsArray = append(flagsArray, "ubuntu cat /inputs/foo.txt") // This doesn't exist, but shouldn't error

		// _, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd,
		// 	flagsArray...,
		// )
		// require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

		// job, _, err := c.Get(ctx, strings.TrimSpace(out))
		// require.NoError(suite.T(), err)
		// require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)

		// require.Equal(suite.T(), len(tcids.inputVolumes), len(job.Spec.Inputs), "Number of job inputs != # of test inputs .")

		// // Need to do the below because ordering is not guaranteed
		// for _, tcidIV := range tcids.inputVolumes {
		// 	testCIDinJobInputs := false
		// 	for _, jobInput := range job.Spec.Inputs {
		// 		if tcidIV.cid == jobInput.Cid {
		// 			testCIDinJobInputs = true
		// 			testPath := "/inputs"
		// 			if tcidIV.path != "" {
		// 				testPath = tcidIV.path
		// 			}
		// 			require.Equal(suite.T(), testPath, jobInput.Path, "Test Path not equal to Path from job.")
		// 			break
		// 		}
		// 	}
		// 	require.True(suite.T(), testCIDinJobInputs, "Test CID not in job inputs.")
		// }
		// }()
		// }
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
			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

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

				args = append(args, "--clear-labels")

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

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDockerRunSuite(t *testing.T) {
	suite.Run(t, new(DockerRunSuite))
}
