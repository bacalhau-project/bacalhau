package bacalhau

import (
	"fmt"
	"net"
	"net/url"
	"strings"
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
		c := publicapi.SetupTests(suite.T())

		// Submit a few random jobs to the node:
		var err error

		parsedBasedURI, _ := url.Parse(c.BaseURI)
		host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
		_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "run",
			"--api-host", host,
			"--api-port", port,
			"ubuntu echo 'hello world'",
		)

		assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

		job, _, err := c.Get(out)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)

		// assert.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))

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
		c := publicapi.SetupTests(suite.T())

		// Submit a few random jobs to the node:
		var err error

		parsedBasedURI, _ := url.Parse(c.BaseURI)
		host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
		_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "run",
			"--api-host", host,
			"--api-port", port,
			"ubuntu echo 'hello world'",
		)

		assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

		job, _, err := c.Get(out)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)

		assert.LessOrEqual(suite.T(), job.CreatedAt, time.Now(), "Created at time is not less than or equal to now.")

		oldStartTime, _ := time.Parse(time.RFC3339, "2021-01-01T01:01:01+00:00")
		assert.GreaterOrEqual(suite.T(), job.CreatedAt, oldStartTime, "Created at time is not greater or equal to 2022-01-01.")

	}
}

func (suite *RunSuite) TestRun_Labels() {

	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		// {numberOfJobs: 5}, // Test for five
	}

	labelsToTest := []struct {
		Labels  []string
		BadCase bool
	}{
		{Labels: []string{""}, BadCase: false},       // Label flag, no value, but correctly quoted
		{Labels: []string{"a"}, BadCase: false},      // Labels, string
		{Labels: []string{"a", "1"}, BadCase: false}, // Labels, string and int
		{Labels: []string{`'`, ` `}, BadCase: false}, // Labels, some edge case characters
	}

	allBadStrings := LoadBadStringsLabels()
	for _, s := range allBadStrings {
		l := struct {
			Labels  []string
			BadCase bool
		}{Labels: []string{s}, BadCase: false}
		labelsToTest = append(labelsToTest, l)
	}

	for i, tc := range tests {
		c := publicapi.SetupTests(suite.T())

		for _, labelTest := range labelsToTest {
			labelString := fmt.Sprintf("\"%s\"", strings.Join(labelTest.Labels, ","))
			var err error
			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "run",
				"--api-host", host,
				"--api-port", port,
				"--labels", labelString,
				"ubuntu echo 'hello world'",
			)

			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			job, _, err := c.Get(out)
			assert.NoError(suite.T(), err)

			if labelTest.BadCase {
				assert.Contains(suite.T(), out, "rror")
			} else {
				assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
				assert.NotContains(suite.T(), out, "rror", "'%s' caused an error", labelTest.Labels)
			}

		}
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(RunSuite))
}
