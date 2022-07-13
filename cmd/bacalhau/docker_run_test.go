package bacalhau

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
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
		{Name: "4", Annotations: []string{`''`, `" "`}, CorrectLength: 2, BadCase: false},      // Annotations, some edge case characters
		{Name: "5", Annotations: []string{"🏳", "0", "🌈️"}, CorrectLength: 3, BadCase: false},   // Emojis
		{Name: "6", Annotations: []string{"ايطاليا"}, CorrectLength: 1, BadCase: false},        // Right to left
		{Name: "7", Annotations: []string{"‫test‫"}, CorrectLength: 1, BadCase: false},         // Control charactel
		{Name: "8", Annotations: []string{"사회과학원", "어학연구소"}, CorrectLength: 2, BadCase: false}, // Two-byte characters
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
		submitString string
		errString    string
	}{
		{submitString: "*.jpg", errString: "contains a glob"},
		{submitString: " /bin/bash *.jpg", errString: ""}, // contains a glob but starts with a shell (and a space)
		// {submitString: "-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images -o results:/output_images dpokidov/imagemagick -- magick mogrify -fx '((g-b)/(r+g+b))>0.02 ? 1 : 0' -resize 256x256 -quality 100 -path /output_images /input_images/*.jpg"},
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
				tc.submitString,
			)
			assert.NoError(suite.T(), err, "Error submitting job. Run - Test-Number: %d - String: %s", i, tc.submitString)

			job, isErr, err := c.Get(ctx, strings.TrimSpace(out))
			assert.False(suite.T(), isErr, "error getting job")
			assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDockerRunSuite(t *testing.T) {
	suite.Run(t, new(DockerRunSuite))
}
