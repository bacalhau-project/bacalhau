package bacalhau

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type RunSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before all suite
func (suite *RunSuite) SetupAllSuite() {

}

// Before each test
func (suite *RunSuite) SetupTest() {
	suite.rootCmd = RootCmd
}

func (suite *RunSuite) TearDownTest() {
}

func (suite *RunSuite) TearDownAllSuite() {

}

func (suite *RunSuite) TestRun_GenericSubmit() {
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
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "run",
				"--api-host", host,
				"--api-port", port,
				"ubuntu echo 'hello world'",
			)
			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, out)
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
			// assert.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))
		}()
	}
}

func (suite *RunSuite) TestRun_CreatedAt() {
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
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "run",
				"--api-host", host,
				"--api-port", port,
				"ubuntu echo 'hello world'",
			)
			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, out)
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
			assert.LessOrEqual(suite.T(), job.CreatedAt, time.Now(), "Created at time is not less than or equal to now.")

			oldStartTime, _ := time.Parse(time.RFC3339, "2021-01-01T01:01:01+00:00")
			assert.GreaterOrEqual(suite.T(), job.CreatedAt, oldStartTime, "Created at time is not greater or equal to 2022-01-01.")
		}()

	}
}

// TODO: #261 Hate to bring this up again, but this is looking like leaking again - non-deterministically failing test in this test.
func (suite *RunSuite) TestRun_Labels() {
	suite.T().Skip("Need to skip due to non-deterministically failing.")

	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		// {numberOfJobs: 5}, // Test for five
	}

	labelsToTest := []struct {
		Labels        []string
		CorrectLength int
		BadCase       bool
	}{
		// {Labels: []string{""}, CorrectLength: 0, BadCase: false},               // Label flag, no value, but correctly quoted
		// {Labels: []string{"a"}, CorrectLength: 1, BadCase: false},              // Labels, string
		// {Labels: []string{"a", "1"}, CorrectLength: 2, BadCase: false},         // Labels, string and int
		{Labels: []string{`'`, ` `}, CorrectLength: 0, BadCase: false},       // Labels, some edge case characters
		{Labels: []string{"🏳", "0", "🌈️"}, CorrectLength: 3, BadCase: false}, // Emojis
		{Labels: []string{"ايطاليا"}, CorrectLength: 1, BadCase: false},      // Right to left
		// {Labels: []string{"‫test‫"}, CorrectLength: 3, BadCase: false},         // Control charactel
		// {Labels: []string{"사회과학원", "어학연구소"}, CorrectLength: 3, BadCase: false}, // Two-byte characters
	}

	// allBadStrings := LoadBadStringsLabels()
	// for _, s := range allBadStrings {
	// 	strippedString := SafeStringStripper(s)
	// 	l := struct {
	// 		Labels        []string
	// 		CorrectLength int
	// 		BadCase       bool
	// 	}{Labels: []string{s}, CorrectLength: len(strippedString), BadCase: false}
	// 	labelsToTest = append(labelsToTest, l)
	// }

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			for _, labelTest := range labelsToTest {
				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, err := net.SplitHostPort(parsedBasedURI.Host)
				assert.NoError(suite.T(), err)

				var args []string
				args = append(args, "run", "--api-host", host, "--api-port", port)
				for _, label := range labelTest.Labels {
					args = append(args, "--labels", label)
				}
				args = append(args, "--clear-labels")
				args = append(args, "ubuntu echo 'hello world'")

				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
				assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				testJob, _, err := c.Get(ctx, out)
				assert.NoError(suite.T(), err)

				if labelTest.BadCase {
					assert.Contains(suite.T(), out, "rror")
				} else {
					assert.NotNil(suite.T(), testJob, "Failed to get job with ID: %s", out)
					assert.NotContains(suite.T(), out, "rror", "'%s' caused an error", labelTest.Labels)
					msg := fmt.Sprintf(`
Number o    f labels stored not equal to expected length.
Expected     length: %d
Actual l    ength: %d

Expected     labels: %+v
Actual l    abels: %+v
`, len(labelTest.Labels), len(testJob.Spec.Labels), labelTest.Labels, testJob.Spec.Labels)

					assert.Equal(suite.T(), len(labelTest.Labels), len(testJob.Spec.Labels), msg)
				}

			}
		}()
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(RunSuite))
}
