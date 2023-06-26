//go:build unit || !integration

package list_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/cmd/cli/list"
	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ListSuite struct {
	cmdtesting.BaseSuite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestListSuite(t *testing.T) {
	suite.Run(t, new(ListSuite))
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
		suite.Run(fmt.Sprintf("%d jobs %d output", tc.numberOfJobs, tc.numberOfJobsOutput), func() {
			ctx := context.Background()

			for i := 0; i < tc.numberOfJobs; i++ {
				j := testutils.MakeNoopJob()
				_, err := suite.Client.Submit(ctx, j)
				require.NoError(suite.T(), err)
			}

			_, out, err := cmdtesting.ExecuteTestCobraCommand("list",
				"--hide-header",
				"--api-host", suite.Host,
				"--api-port", fmt.Sprint(suite.Port),
				"--number", fmt.Sprintf("%d", tc.numberOfJobsOutput),
				"--reverse", "false",
			)
			require.NoError(suite.T(), err)

			require.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))
		})
	}
}

func (suite *ListSuite) TestList_IdFilter() {
	ctx := context.Background()

	// submit 10 jobs
	var jobIds []string
	var jobLongIds []string
	for i := 0; i < 10; i++ {
		var err error
		j := testutils.MakeNoopJob()
		j, err = suite.Client.Submit(ctx, j)
		jobIds = append(jobIds, system.GetShortID(j.Metadata.ID))
		jobLongIds = append(jobIds, j.Metadata.ID)
		require.NoError(suite.T(), err)
	}
	_, out, err := cmdtesting.ExecuteTestCobraCommand("list",
		"--hide-header",
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
		"--id-filter", jobIds[0],
		"--wide",
	)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "\n", out[len(out)-1:], "Expected output to end with a newline: %q", out)

	// parse list output
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Equal(suite.T(), 1, len(lines), "We didn't get only one result: %q", lines)
	require.Contains(suite.T(), lines[0], jobIds[0], "The returned job id was not what we asked for")

	//// Test --output json

	// _, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "list",
	_, out, err = cmdtesting.ExecuteTestCobraCommand("list",
		"--hide-header",
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
		"--id-filter", jobLongIds[0],
		"--output", "json",
	)
	require.NoError(suite.T(), err)

	// parse response
	response := publicapi.ListResponse{}
	err = model.JSONUnmarshalWithMax([]byte(out), &response.Jobs)

	var firstItem model.Job
	for _, v := range response.Jobs {
		firstItem = v.Job
		break
	}

	require.NoError(suite.T(), err)

	require.Contains(suite.T(), firstItem.Metadata.ID, jobLongIds[0], "The filtered job id was not found in the response")
	require.Equal(suite.T(), 1, len(response.Jobs), "The list of jobs is not strictly filtered to the requested job id")
}

func (suite *ListSuite) TestList_AnnotationFilter() {
	type testCase struct {
		Name                                              string
		JobLabels, ListLabels                             []string
		AppearByDefault, AppearOnInclude, AppearOnExclude bool
	}

	testCases := []testCase{
		{"empty filters have no effect", []string{}, []string{}, true, true, true},
		{"include filters unlabelled jobs", []string{}, []string{"test"}, true, false, true},
		{"exclude filters labelled jobs", []string{"test"}, []string{"test"}, true, true, false},
		{"filters match job labels", []string{"jobb"}, []string{"test"}, true, false, true},
		{"multiple annotations match any", []string{"test", "jobb"}, []string{"test"}, true, true, false},
		{"multiple filters match any", []string{"t1"}, []string{"t1", "t2"}, true, true, false},
	}

	for _, tag := range list.DefaultExcludedTags {
		testCases = append(testCases, testCase{
			fmt.Sprintf("%s filtered by default", string(tag)),
			[]string{string(tag)},
			[]string{string(tag)},
			false,
			true,
			false,
		})
		testCases = append(testCases, testCase{
			fmt.Sprintf("%s excluded with other tags", string(tag)),
			[]string{string(tag)},
			[]string{"test"},
			false,
			false,
			false,
		})
	}

	for _, tc := range testCases {
		suite.Run(tc.Name, func() {
			ctx := context.Background()
			// have to create a fresh node for each test case to avoid jobs of different runs to be mixed up
			suite.TearDownTest()
			suite.SetupTest()

			j := testutils.MakeNoopJob()
			j.Spec.Annotations = tc.JobLabels
			j, err := suite.Client.Submit(ctx, j)
			require.NoError(suite.T(), err)

			checkList := func(shouldAppear bool, flags ...string) {
				args := []string{"list",
					"--hide-header",
					"--api-host", suite.Host,
					"--api-port", fmt.Sprint(suite.Port),
					"--output", "json",
				}
				args = append(args, flags...)
				_, out, err := cmdtesting.ExecuteTestCobraCommand(args...)
				require.NoError(suite.T(), err)

				response := publicapi.ListResponse{}
				_ = model.JSONUnmarshalWithMax([]byte(out), &response.Jobs)
				if shouldAppear {
					require.NotEmpty(suite.T(), response.Jobs)
					require.Equal(suite.T(), j.Metadata.ID, response.Jobs[0].Job.Metadata.ID)
				} else {
					require.Empty(suite.T(), response.Jobs)
				}
			}

			// default list
			suite.Run("default", func() {
				checkList(tc.AppearByDefault)
			})

			// list with label included
			suite.Run("label_included", func() {
				var flags []string
				for _, label := range tc.ListLabels {
					flags = append(flags, "--include-tag", label)
				}
				checkList(tc.AppearOnInclude, flags...)
			})

			// list with label excluded
			suite.Run("label_excluded", func() {
				var flags []string
				for _, label := range tc.ListLabels {
					flags = append(flags, "--exclude-tag", label)
				}
				checkList(tc.AppearOnExclude, flags...)
			})
		})
	}
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
		{sortFlag: string(list.ColumnID), reverseFlag: false},
		{sortFlag: string(list.ColumnID), reverseFlag: true},
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
					j, err = suite.Client.Submit(ctx, j)
					require.NoError(suite.T(), err)
					jobIDs = append(jobIDs, system.GetShortID(j.Metadata.ID))

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

				_, out, err := cmdtesting.ExecuteTestCobraCommand("list",
					"--hide-header",
					"--no-style",
					"--api-host", suite.Host,
					"--api-port", fmt.Sprint(suite.Port),
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
						if sortFlags.sortFlag == string(list.ColumnID) {
							sort.Strings(jobIDs)
						}

						if sortFlags.reverseFlag {
							jobIDs = system.ReverseList(jobIDs)
						}

						compareIds := jobIDs[0:tc.numberOfJobsOutput]
						var seenIds []string

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

						if sortFlags.sortFlag == string(list.ColumnID) {
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
