package bacalhau

import (
	"fmt"
	"net"
	"net/url"
	"reflect"
	"sort"
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
	tableIDFilter = ""
	tableSortReverse = false

	tests := []struct {
		numberOfJobs       int
		numberOfJobsOutput int
	}{
		{numberOfJobs: 0, numberOfJobsOutput: 0},   // Test for zero
		{numberOfJobs: 5, numberOfJobsOutput: 5},   // Test for 5 (less than default of 10)
		{numberOfJobs: 20, numberOfJobsOutput: 10}, // Test for 20 (more than max of 10)
		{numberOfJobs: 20, numberOfJobsOutput: 15}, // The default is 10 so test for non-default

	}

	for _, tc := range tests {
		ctx, c := publicapi.SetupTests(suite.T())

		// Submit a few random jobs to the node:
		var err error

		for i := 0; i < tc.numberOfJobs; i++ {
			spec, deal := publicapi.MakeNoopJob()
			_, err = c.Submit(ctx, spec, deal)
			assert.NoError(suite.T(), err)
		}

		parsedBasedURI, _ := url.Parse(c.BaseURI)
		host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
		_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "list",
			"--hide-header",
			"--api-host", host,
			"--api-port", port,
			"--number", fmt.Sprintf("%d", tc.numberOfJobsOutput),
		)

		assert.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))

	}
}

func (suite *ListSuite) TestList_IdFilter() {
	ctx, c := publicapi.SetupTests(suite.T())

	jobIds := []string{}
	for i := 0; i < 10; i++ {
		spec, deal := publicapi.MakeNoopJob()
		job, err := c.Submit(ctx, spec, deal)
		jobIds = append(jobIds, shortID(job.ID))
		assert.NoError(suite.T(), err)
	}

	parsedBasedURI, _ := url.Parse(c.BaseURI)
	host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
	_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "list",
		"--hide-header",
		"--api-host", host,
		"--api-port", port,
		"--id-filter", jobIds[0],
	)
	assert.NoError(suite.T(), err)

	seenIds := []string{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, " ")
		if len(parts) > 2 {
			seenIds = append(seenIds, strings.Split(line, " ")[1])
		}
	}

	assert.Equal(suite.T(), 1, len(seenIds), "We didn't get only one result")
	assert.Equal(suite.T(), seenIds[0], jobIds[0], "The returned job id was not what we asked for")
}

func (suite *ListSuite) TestList_SortFlags() {
	var badSortFlag = "BADSORTFLAG"
	var createdAtSortFlag = "created_at"
	tableIDFilter = ""
	tableSortReverse = false

	combinationOfJobSizes := []struct {
		numberOfJobs       int
		numberOfJobsOutput int
	}{
		{numberOfJobs: 0, numberOfJobsOutput: 0},   // Test for zero
		{numberOfJobs: 5, numberOfJobsOutput: 5},   // Test for 5 (less than default of 10)
		{numberOfJobs: 20, numberOfJobsOutput: 10}, // Test for 20 (more than max of 10)
		{numberOfJobs: 20, numberOfJobsOutput: 15}, // The default is 10 so test for non-default
	}

	sortFlagsToTest := []struct {
		sortFlag    string
		reverseFlag bool
		badSortFlag bool
	}{
		{sortFlag: string(ColumnID), reverseFlag: false},
		{sortFlag: string(ColumnID), reverseFlag: true},
		{sortFlag: createdAtSortFlag, reverseFlag: false},
		{sortFlag: createdAtSortFlag, reverseFlag: true},
		{sortFlag: badSortFlag, reverseFlag: false, badSortFlag: true},
		{sortFlag: badSortFlag, reverseFlag: true, badSortFlag: true},
		{sortFlag: "", reverseFlag: false, badSortFlag: true},
		{sortFlag: "", reverseFlag: true, badSortFlag: true},
	}

	for _, tc := range combinationOfJobSizes {
		for _, sortFlags := range sortFlagsToTest {
			ctx, c := publicapi.SetupTests(suite.T())

			// Submit a few random jobs to the node:
			var err error

			jobIds := []string{}

			for i := 0; i < tc.numberOfJobs; i++ {
				spec, deal := publicapi.MakeNoopJob()
				job, err := c.Submit(ctx, spec, deal)
				assert.NoError(suite.T(), err)
				jobIds = append(jobIds, shortID(job.ID))

				// all the middle jobs can have the same timestamp
				// but we need the first and last to differ
				// so we can test sorting on time stamp
				if (i == 0 || i == tc.numberOfJobs-2) && sortFlags.sortFlag == createdAtSortFlag {
					time.Sleep(1 * time.Second)
				} else {
					time.Sleep(1 * time.Millisecond)
				}
			}

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)

			reverseString := ""
			if sortFlags.reverseFlag {
				reverseString = "--reverse"
			}

			// IMPORTANT: reset this to the default value because otherwise strange things happen
			// between tests because the value is held on from the last CLI invocation
			tableSortReverse = false

			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd,
				"list",
				"--hide-header",
				"--no-style",
				"--api-host", host,
				"--api-port", port,
				"--sort-by", sortFlags.sortFlag,
				"--number", fmt.Sprintf("%d", tc.numberOfJobsOutput),
				reverseString,
			)

			if sortFlags.badSortFlag {
				assert.Error(suite.T(), err, "No error was thrown though it was a bad sort flag: %s", badSortFlag)
				assert.Contains(suite.T(), out, "Error: invalid argument", "'--sort-by' did not reject bad sort flag: %s", badSortFlag)
			} else {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))

				if tc.numberOfJobsOutput > 0 {

					// jobIds are already sorted by created ASC
					if sortFlags.sortFlag == string(ColumnID) {
						sort.Strings(jobIds)
					}

					if sortFlags.reverseFlag {
						jobIds = ReverseList(jobIds)
					}

					compareIds := jobIds[0:tc.numberOfJobsOutput]
					seenIds := []string{}

					for _, line := range strings.Split(out, "\n") {
						parts := strings.Split(line, " ")
						if len(parts) > 2 {
							seenIds = append(seenIds, strings.Split(line, " ")[1])
						}
					}

					errorMessage := fmt.Sprintf(`
Table lines do not match
Number of Jobs: %d
Number of Max: %d
Sort Flag: %s
Reverse Flag: %t

Out:
%s

Seen Ids:
%s

Compare Ids:
%s

					`, tc.numberOfJobs, tc.numberOfJobsOutput, sortFlags.sortFlag, sortFlags.reverseFlag, out, strings.Join(seenIds, " "), strings.Join(compareIds, " "))

					if sortFlags.sortFlag == string(ColumnID) {
						assert.True(suite.T(), reflect.DeepEqual(compareIds, seenIds), errorMessage)
					} else if sortFlags.sortFlag == createdAtSortFlag {
						// check the first and last are correct
						// the middles all have the same created time so we ignore those
						assert.Equal(suite.T(), compareIds[0], seenIds[0], errorMessage)
					}
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
