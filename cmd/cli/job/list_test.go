//go:build integration || !unit

package job_test

import (
	"encoding/csv"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestListSuite(t *testing.T) {
	suite.Run(t, new(ListSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ListSuite struct {
	cmdtesting.BaseSuite
}

// Before each test
func (s *ListSuite) SetupTest() {
	docker.MustHaveDocker(s.T())
	s.BaseSuite.SetupTest()
}

func (s *ListSuite) TestJobList() {
	// Submit a job
	args := []string{
		"docker", "run",
		"--wait",
		"alpine", "echo", "hello world",
	}
	_, out, err := s.ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)

	// Run 'bacalhau job list'
	_, listOut, err := s.ExecuteTestCobraCommand("job", "list")
	require.NoError(s.T(), err, "Error running job list")

	// Check that the shortened job ID appears in the list
	require.Contains(s.T(), listOut, jobID[:10], "Short Job ID not found in list output")
}

func (s *ListSuite) TestJobListLimit() {
	// Submit multiple jobs
	numJobs := 5
	jobIDs := make([]string, numJobs)
	for i := 0; i < numJobs; i++ {
		args := []string{
			"docker", "run",
			"--wait",
			"alpine", "echo", fmt.Sprintf("hello world %d", i),
		}
		_, out, err := s.ExecuteTestCobraCommand(args...)
		require.NoError(s.T(), err, "Error submitting job")
		jobID := system.FindJobIDInTestOutput(out)
		jobIDs[i] = jobID
	}

	// Run 'bacalhau job list --limit 2 --output csv'
	_, listOut, err := s.ExecuteTestCobraCommand("job", "list", "--wide", "--limit", "2", "--output", "csv")
	require.NoError(s.T(), err, "Error running job list")

	// Extract the CSV data from the output
	csvData, remainingOutput, err := system.ExtractCSVData(listOut)
	require.NoError(s.T(), err, "Error extracting CSV data from output")
	require.NotEmpty(s.T(), csvData, "CSV data is empty")

	// Parse the CSV data
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	require.NoError(s.T(), err, "Error parsing CSV output")

	// The first record is the header
	require.Greater(s.T(), len(records), 1, "Expected at least one job in list output")
	headers := records[0]
	jobIDIndex := -1
	for i, header := range headers {
		if header == "id" {
			jobIDIndex = i
			break
		}
	}
	require.NotEqual(s.T(), -1, jobIDIndex, "Job ID column not found in CSV headers")

	// Extract job IDs from the CSV records
	var jobs []string
	for _, record := range records[1:] { // Skip header
		if len(record) > jobIDIndex {
			jobs = append(jobs, record[jobIDIndex])
		}
	}

	// Check that exactly 2 jobs are returned
	require.Equal(s.T(), 2, len(jobs), "Expected 2 jobs in list output")

	expectedJobIDs := jobIDs[:2]

	// Check that the job IDs match the expected values
	for i, jobID := range jobs {
		require.Equal(s.T(), expectedJobIDs[i], jobID, "Job ID does not match expected")
	}

	// Verify that the message about fetching more records is present
	expectedMessage := fmt.Sprintf("To fetch more records use:\n\tbacalhau job list --limit %d --next-token", 2)
	require.Contains(s.T(), remainingOutput, expectedMessage, "Pagination message not found in output")
}
