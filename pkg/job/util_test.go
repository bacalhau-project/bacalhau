//go:build unit || !integration

package job

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	storagetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestJobUtilSuite(t *testing.T) {
	suite.Run(t, new(JobUtilSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type JobUtilSuite struct {
	suite.Suite
}

// Before each test
func (s *JobUtilSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
}

func (s *JobUtilSuite) TestRun_URLs() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for range tests {
		testURLs := []struct {
			submittedURL string
			convertedURL string // if we parse it, this is what it should look like
			valid        bool
			errorMsg     string
		}{
			{submittedURL: "http://example.com",
				valid:    false,
				errorMsg: "TYPE: Invalid (no file)"},
			{submittedURL: "http://example.com/file.txt",
				valid:    true,
				errorMsg: "TYPE: Valid"},
			{submittedURL: "ttps://example.com",
				valid:    false,
				errorMsg: "TYPE: Bad scheme"},
			{submittedURL: "example.com",
				valid:    false,
				errorMsg: "TYPE: Mising scheme"},
			{submittedURL: "http://example.com:8080/file.txt",
				valid:    true,
				errorMsg: "TYPE: With Ports"},
			{submittedURL: `https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD`,
				valid:    true,
				errorMsg: "TYPE: With query string"},
			{submittedURL: `https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar`,
				valid:    true,
				errorMsg: "TYPE: With query string with ampersand"},
			{submittedURL: `"https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar"`,
				valid:    true,
				errorMsg: "TYPE: With Double quotes"},
			{submittedURL: `'https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar'`,
				valid:    true,
				errorMsg: "TYPE: With single quotes"},
		}

		for _, testURL := range testURLs {
			func() {
				// Test all URLs against the validator
				spec, err := ParseStorageString(testURL.submittedURL, "/inputs", map[string]string{})
				originalURLTrimmed := strings.Trim(testURL.submittedURL, `"' `)
				convertedTrimmed := strings.Trim(testURL.convertedURL, `"' `)
				if testURL.valid {
					require.NoError(s.T(), err, fmt.Sprintf("%s: Should not have errored - %s", testURL.errorMsg, testURL.submittedURL))
					if testURL.convertedURL != "" {
						require.Equal(s.T(), convertedTrimmed, storagetesting.URLDecodeStorage(s.T(), spec).URL, testURL.errorMsg)
					} else {
						require.Equal(s.T(), originalURLTrimmed, storagetesting.URLDecodeStorage(s.T(), spec).URL, testURL.errorMsg)
					}
				} else {
					require.Error(s.T(), err, fmt.Sprintf("%s: Should have errored - %s", testURL.errorMsg, testURL.submittedURL))
				}
			}()
		}
	}
}

func (s *JobUtilSuite) TestStateSummary() {

	tc := []struct {
		name     string
		states   []model.ExecutionStateType
		expected string
	}{
		{
			name: "All Rejected",
			states: []model.ExecutionStateType{
				model.ExecutionStateBidRejected,
				model.ExecutionStateBidRejected,
				model.ExecutionStateBidRejected,
			},
			expected: "BidRejected",
		},
		{
			name: "Accepted Bid in minority report",
			states: []model.ExecutionStateType{
				model.ExecutionStateBidAccepted,
				model.ExecutionStateBidRejected,
				model.ExecutionStateBidRejected,
			},
			expected: "BidAccepted",
		},
		{
			name: "Completed wins out against rejected bids",
			states: []model.ExecutionStateType{
				model.ExecutionStateCompleted,
				model.ExecutionStateAskForBidAccepted,
				model.ExecutionStateAskForBidRejected,
			},
			expected: "Completed",
		},
		{
			name: "Canceled wins out against Bid Accepted and Rejected",
			states: []model.ExecutionStateType{
				model.ExecutionStateCanceled,
				model.ExecutionStateAskForBidAccepted,
				model.ExecutionStateAskForBidRejected,
			},
			expected: "Cancelled",
		},
	}

	for _, testCase := range tc {
		s.Run(testCase.name, func() {
			j := model.JobState{
				Executions: []model.ExecutionState{
					{State: testCase.states[0]},
					{State: testCase.states[1]},
					{State: testCase.states[2]},
				},
			}

			summary := ComputeStateSummary(j)
			require.Equal(s.T(), testCase.expected, summary)
		})
	}
}
