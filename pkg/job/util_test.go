package job

import (
	"fmt"
	"testing"

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

// Before all suite
func (s *JobUtilSuite) SetupAllSuite() {

}

// Before each test
func (s *JobUtilSuite) SetupTest() {
}

func (s *JobUtilSuite) TearDownTest() {
}

func (s *JobUtilSuite) TearDownAllSuite() {

}

func (s *JobUtilSuite) TestRun_URLs() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for range tests {
		testURLs := []struct {
			uri        string
			uriResult  string // if we parse it, this is what it should look like
			pathResult string
			valid      bool
			errorMsg   string
		}{
			{uri: "http://example.com",
				valid:    true,
				errorMsg: "TYPE: Valid"},
			{uri: "ttps://example.com",
				valid:    false,
				errorMsg: "TYPE: Bad scheme"},
			{uri: "example.com",
				valid:    false,
				errorMsg: "TYPE: Mising scheme"},
			{uri: "http://example.com:8080",
				valid:    true,
				errorMsg: "TYPE: With Ports"},
			{uri: "http://example.com:8080/",
				uriResult: "http://example.com:8080",
				valid:     true,
				errorMsg:  "TYPE: With Ports and trailing slash"},
			{uri: `https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD`,
				uriResult:  "https://data.cityofnewyork.us",
				pathResult: "/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD",
				valid:      true,
				errorMsg:   "TYPE: With query string"},
			{uri: `https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar`,
				uriResult:  "https://data.cityofnewyork.us",
				pathResult: "/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar",
				valid:      true,
				errorMsg:   "TYPE: With query string with ampersand"},
			{uri: `"https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar"`,
				uriResult:  "https://data.cityofnewyork.us",
				pathResult: "/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar",
				valid:      true,
				errorMsg:   "TYPE: With Double quotes"},
			{uri: `'https://data.cityofnewyork.us/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar'`,
				uriResult:  "https://data.cityofnewyork.us",
				pathResult: "/api/views/t29m-gskq/rows.csv?accessType=DOWNLOAD&foo=bar",
				valid:      true,
				errorMsg:   "TYPE: With single quotes"},
		}

		for _, testURL := range testURLs {
			func() {
				// Test all URLs against the validator
				spec, err := buildJobInputs(nil, []string{testURL.uri})
				if testURL.valid {
					require.NoError(s.T(), err, fmt.Sprintf("%s: Should not have errored - %s", testURL.errorMsg, testURL.uri))
					require.Equal(s.T(), 1, len(spec), testURL.errorMsg)
					if testURL.uriResult != "" {
						require.Equal(s.T(), testURL.uriResult, spec[0].URL, testURL.errorMsg)
					} else {
						require.Equal(s.T(), testURL.uri, spec[0].URL, testURL.errorMsg)
					}
					if testURL.pathResult != "" {
						require.Equal(s.T(), testURL.pathResult, spec[0].Path, testURL.errorMsg)
					}
				} else {
					require.Error(s.T(), err, fmt.Sprintf("%s: Should have errored - %s", testURL.errorMsg, testURL.uri))
				}
			}()
		}
	}
}
