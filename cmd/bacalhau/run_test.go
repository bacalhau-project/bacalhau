package bacalhau

import (
	"context"
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
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "docker", "run",
				"--api-host", host,
				"--api-port", port,
				"ubuntu echo 'hello world'",
			)
			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
			assert.LessOrEqual(suite.T(), job.CreatedAt, time.Now(), "Created at time is not less than or equal to now.")

			oldStartTime, _ := time.Parse(time.RFC3339, "2021-01-01T01:01:01+00:00")
			assert.GreaterOrEqual(suite.T(), job.CreatedAt, oldStartTime, "Created at time is not greater or equal to 2022-01-01.")
		}()

	}
}
func (suite *RunSuite) TestRun_Annotations() {
	// TODO: Assign to Aronchick - change tests to just reject if annotation is bad.
	suite.T().Skip()

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
		{Name: "1.1", Annotations: []string{`""`}, CorrectLength: 1, BadCase: false},           // Label flag, no value, but correctly quoted
		{Name: "2", Annotations: []string{"a"}, CorrectLength: 1, BadCase: false},              // Annotations, string
		{Name: "3", Annotations: []string{"a", "1"}, CorrectLength: 2, BadCase: false},         // Annotations, string and int
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
				parsedBasedURI, err := url.Parse(c.BaseURI)
				assert.NoError(suite.T(), err)

				host, port, err := net.SplitHostPort(parsedBasedURI.Host)
				assert.NoError(suite.T(), err)

				var args []string
				args = append(args, "docker", "run", "--api-host", host, "--api-port", port)
				for _, label := range labelTest.Annotations {
					args = append(args, "annotations", label)
				}
				args = append(args, "--clear-annotations")
				args = append(args, "ubuntu echo 'hello world'")

				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
				assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				testJob, _, err := c.Get(ctx, strings.TrimSpace(out))
				assert.NoError(suite.T(), err)

				if labelTest.BadCase {
					assert.Contains(suite.T(), out, "rror")
				} else {
					assert.NotNil(suite.T(), testJob, "Failed to get job with ID: %s", out)
					assert.NotContains(suite.T(), out, "rror", "'%s' caused an error", labelTest.Annotations)
					msg := fmt.Sprintf(`
Number of Annotations stored not equal to expected length.
Name: %s
Expected length: %d
Actual length: %d

Expected Annotations: %+v
Actual Annotations: %+v
`, labelTest.Name, len(labelTest.Annotations), len(testJob.Spec.Annotations), labelTest.Annotations, testJob.Spec.Annotations)
					assert.Equal(suite.T(), labelTest.CorrectLength, len(testJob.Spec.Annotations), msg)
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
