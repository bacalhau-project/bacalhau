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
type ListSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before all suite
func (suite *ListSuite) SetupAllSuite() {

}

// Before each test
func (suite *ListSuite) SetupTest() {
	suite.rootCmd = RootCmd
}

func (suite *ListSuite) TearDownTest() {
}

func (suite *ListSuite) TearDownAllSuite() {

}

func (suite *ListSuite) TestList_NumberOfJobs() {
	tests := []struct {
		numberOfJobs       int
		numberOfJobsOutput int
	}{
		{numberOfJobs: 0, numberOfJobsOutput: 0},   // Test for zero
		{numberOfJobs: 5, numberOfJobsOutput: 5},   // Test for 5 (less than default of 10)
		{numberOfJobs: 20, numberOfJobsOutput: 10}, // Test for 20 (more than max of 10)
	}

	for _, tc := range tests {
		c := publicapi.SetupTests(suite.T())

		// Submit a few random jobs to the node:
		var err error

		for i := 0; i < tc.numberOfJobs; i++ {
			_, err = c.Submit(publicapi.MakeNoopJob())
			assert.NoError(suite.T(), err)
		}

		parsedBasedURI, _ := url.Parse(c.BaseURI)
		host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
		_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "list", "--api-host", host, "--api-port", port, "--hide-header")

		assert.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))

	}
}

func (suite *ListSuite) TestList_SortFlags() {
	var badSortFlag = "BADSORTFLAG"

	combinationOfJobSizes := []struct {
		numberOfJobs       int
		numberOfJobsOutput int
	}{
		// {numberOfJobs: 0, numberOfJobsOutput: 0},   // Test for zero
		{numberOfJobs: 5, numberOfJobsOutput: 5},   // Test for 5 (less than default of 10)
		{numberOfJobs: 20, numberOfJobsOutput: 10}, // Test for 20 (more than max of 10)
	}

	sortFlagsToTest := []struct {
		sortFlag    string
		reverseFlag bool
		badSortFlag bool
	}{
		{sortFlag: string(ColumnID), reverseFlag: false},
		{sortFlag: string(ColumnID), reverseFlag: true},
		// {sortFlag: "created_at", reverseFlag: false},
		// {sortFlag: "created_at", reverseFlag: true},
		// {sortFlag: badSortFlag, reverseFlag: false, badSortFlag: true},
		// {sortFlag: badSortFlag, reverseFlag: true, badSortFlag: true},
		// {sortFlag: "", reverseFlag: false, badSortFlag: true},
		// {sortFlag: "", reverseFlag: true, badSortFlag: true},
	}

	for _, tc := range combinationOfJobSizes {
		for _, sortFlags := range sortFlagsToTest {
			c := publicapi.SetupTests(suite.T())

			// Submit a few random jobs to the node:
			var err error

			// Collect the first and last job ids for time stamped sorting comparison
			var firstJobId = "ffffffff"
			var lastJobId = "00000000"

			for i := 0; i < tc.numberOfJobs; i++ {
				job, err := c.Submit(publicapi.MakeNoopJob())
				if sortFlags.sortFlag == string(ColumnID) {
					if job.Id < firstJobId {
						firstJobId = job.Id
					}
					if job.Id > lastJobId {
						lastJobId = job.Id
					}
				} else {
					if i == 0 {
						firstJobId = job.Id
					}
					lastJobId = job.Id

					// Need to sleep for at least one second between first and last jobs (otherwise we can't sort)
					time.Sleep(1 * time.Second / time.Duration(tc.numberOfJobs))
				}
				assert.NoError(suite.T(), err)
			}

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)

			reverseString := ""
			if sortFlags.reverseFlag {
				reverseString = "--reverse"
			}
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "list", "--api-host", host, "--api-port", port, "--hide-header", "--no-style", "--sort-by", sortFlags.sortFlag, reverseString)

			if sortFlags.badSortFlag {
				assert.Error(suite.T(), err, "No error was thrown though it was a bad sort flag: %s", badSortFlag)
				assert.Contains(suite.T(), out, "Error: invalid argument", "'--sort-by' did not reject bad sort flag: %s", badSortFlag)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))

				if tc.numberOfJobsOutput > 0 {
					firstLine := strings.Split(out, "\n")[0]
					var idToCompare string

					if sortFlags.reverseFlag {
						idToCompare = shortId(lastJobId)
					} else {
						idToCompare = shortId(firstJobId)
					}

					errorMessage := fmt.Sprintf(`
First line of the return table does not contain the expected ID.
First line: %s
ID: %s
Number of Jobs: %d
Number of Max: %d
Sort Flag: %s
Reverse Flag: %t

Out:
%s
`, firstLine, idToCompare, tc.numberOfJobs, tc.numberOfJobsOutput, sortFlags.sortFlag, sortFlags.reverseFlag, out)

					assert.Contains(suite.T(), firstLine, idToCompare, errorMessage)
				}
			}
		}
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestListSuite(t *testing.T) {
	suite.Run(t, new(ListSuite))
}
