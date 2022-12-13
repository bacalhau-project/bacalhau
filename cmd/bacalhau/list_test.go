//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ListSuite struct {
	BaseSuite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestListSuite(t *testing.T) {
	suite.Run(t, new(ListSuite))
}

type listResponse struct {
	Jobs []*model.Job `json:"jobs"`
}

func (suite *ListSuite) TestList_NumberOfJobs() {
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
		func() {
			ctx := context.Background()

			for i := 0; i < tc.numberOfJobs; i++ {
				j := testutils.MakeNoopJob()
				_, err := suite.client.Submit(ctx, j, nil)
				require.NoError(suite.T(), err)
			}

			_, out, err := ExecuteTestCobraCommand(suite.T(), "list",
				"--hide-header",
				"--api-host", suite.host,
				"--api-port", suite.port,
				"--number", fmt.Sprintf("%d", tc.numberOfJobsOutput),
				"--reverse", "false",
			)
			require.NoError(suite.T(), err)

			require.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))
		}()
	}
}

func (suite *ListSuite) TestList_IdFilter() {
	ctx := context.Background()

	// submit 10 jobs
	jobIds := []string{}
	jobLongIds := []string{}
	for i := 0; i < 10; i++ {
		var err error
		j := testutils.MakeNoopJob()
		j, err = suite.client.Submit(ctx, j, nil)
		jobIds = append(jobIds, shortID(false, j.ID))
		jobLongIds = append(jobIds, j.ID)
		require.NoError(suite.T(), err)
	}
	_, out, err := ExecuteTestCobraCommand(suite.T(), "list",
		"--hide-header",
		"--api-host", suite.host,
		"--api-port", suite.port,
		"--id-filter", jobIds[0],
	)
	require.NoError(suite.T(), err)

	// parse list output
	seenIds := []string{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, " ")
		if len(parts) > 2 {
			seenIds = append(seenIds, strings.Split(line, " ")[3])
		}
	}

	require.Equal(suite.T(), 1, len(seenIds), "We didn't get only one result")
	require.Equal(suite.T(), seenIds[0], jobIds[0], "The returned job id was not what we asked for")

	//// Test --output json

	// _, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "list",
	_, out, err = ExecuteTestCobraCommand(suite.T(), "list",
		"--hide-header",
		"--api-host", suite.host,
		"--api-port", suite.port,
		"--id-filter", jobLongIds[0],
		"--output", "json",
	)
	require.NoError(suite.T(), err)

	// parse response
	response := listResponse{}
	err = model.JSONUnmarshalWithMax([]byte(out), &response.Jobs)

	var firstItem *model.Job
	for _, v := range response.Jobs {
		firstItem = v
		break
	}

	require.NoError(suite.T(), err)

	require.Contains(suite.T(), firstItem.ID, jobLongIds[0], "The filtered job id was not found in the response")
	require.Equal(suite.T(), 1, len(response.Jobs), "The list of jobs is not strictly filtered to the requested job id")
}

func (suite *ListSuite) TestList_SortFlags() {
	var badSortFlag = "BADSORTFLAG"
	var createdAtSortFlag = "created_at"

	combinationOfJobSizes := []struct {
		numberOfJobs       int
		numberOfJobsOutput int
	}{
		// {numberOfJobs: 0, numberOfJobsOutput: 0},   // Test for zero
		{numberOfJobs: 5, numberOfJobsOutput: 5}, // Test for 5 (less than default of 10)
		// {numberOfJobs: 20, numberOfJobsOutput: 10}, // Test for 20 (more than max of 10)
		// {numberOfJobs: 20, numberOfJobsOutput: 15}, // The default is 10 so test for non-default
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
			suite.Run(fmt.Sprintf("%+v/%+v", tc, sortFlags), func() {
				ctx := context.Background()

				// have to create a fresh node for each test case to avoid jobs of different runs to be mixed up
				suite.TearDownTest()
				suite.SetupTest()

				var jobIDs []string
				for i := 0; i < tc.numberOfJobs; i++ {
					var err error
					j := testutils.MakeNoopJob()
					j, err = suite.client.Submit(ctx, j, nil)
					require.NoError(suite.T(), err)
					jobIDs = append(jobIDs, shortID(false, j.ID))

					// all the middle jobs can have the same timestamp
					// but we need the first and last to differ
					// so we can test sorting on time stamp
					if (i == 0 || i == tc.numberOfJobs-2) && sortFlags.sortFlag == createdAtSortFlag {
						time.Sleep(1 * time.Second)
					} else {
						time.Sleep(1 * time.Millisecond)
					}
				}
				reverseString := "--reverse=false"
				if sortFlags.reverseFlag {
					reverseString = "--reverse"
				}

				_, out, err := ExecuteTestCobraCommand(suite.T(),
					"list",
					"--hide-header",
					"--no-style",
					"--api-host", suite.host,
					"--api-port", suite.port,
					"--sort-by", sortFlags.sortFlag,
					"--number", fmt.Sprintf("%d", tc.numberOfJobsOutput),
					reverseString,
				)

				if sortFlags.badSortFlag {
					require.Error(suite.T(), err, "No error was thrown though it was a bad sort flag: %s", badSortFlag)
					require.Contains(suite.T(), out, "Error: invalid argument", "'--sort-by' did not reject bad sort flag: %s", badSortFlag)
				} else {
					require.NoError(suite.T(), err)
					require.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))

					if tc.numberOfJobsOutput > 0 {

						// jobIDs are already sorted by created ASC
						if sortFlags.sortFlag == string(ColumnID) {
							sort.Strings(jobIDs)
						}

						if sortFlags.reverseFlag {
							jobIDs = system.ReverseList(jobIDs)
						}

						compareIds := jobIDs[0:tc.numberOfJobsOutput]
						seenIds := []string{}

						for _, line := range strings.Split(out, "\n") {
							parts := strings.Split(line, " ")
							if len(parts) > 2 {
								seenIds = append(seenIds, strings.Split(line, " ")[3])
							}
						}

						errorMessage := fmt.Sprintf(`
Table lines do not match
Number of Jobs: %d
Number of Max Jobs: %d
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
							require.True(suite.T(), reflect.DeepEqual(compareIds, seenIds), errorMessage)
						} else if sortFlags.sortFlag == createdAtSortFlag {
							// check the first and last are correct
							// the middles all have the same created time so we ignore those
							require.Equal(suite.T(), compareIds[0], seenIds[0], errorMessage)
						}
					}
				}
			})
		}
	}
}
