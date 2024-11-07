//go:build integration || !unit

package job_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	s.Require().NoError(err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)

	// Run 'bacalhau job list'
	_, listOut, err := s.ExecuteTestCobraCommand("job", "list")
	s.Require().NoError(err, "Error running job list")

	// Check that the shortened job ID appears in the list
	s.Require().Contains(listOut, jobID[:10], "Short Job ID not found in list output")
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
		s.Require().NoError(err, "Error submitting job")
		jobID := system.FindJobIDInTestOutput(out)
		jobIDs[i] = jobID
	}

	// Run 'bacalhau job list --limit 2 --output json'
	_, listOut, err := s.ExecuteTestCobraCommand("job", "list", "--wide", "--limit", "2", "--output", "json")
	s.Require().NoError(err, "Error running job list")

	// Extract the JSON data from the output
	jsonData, remainingOutput, err := ExtractJSONOutput(listOut)
	s.Require().NoError(err, "Error extracting JSON data from output")
	s.Require().NotEmpty(jsonData, "JSON data is empty")

	// Parse the JSON data
	var jobs []*models.Job
	err = json.Unmarshal([]byte(jsonData), &jobs)
	s.Require().NoError(err, "Error parsing JSON output")

	// Check that exactly 2 jobs are returned
	s.Require().Equal(2, len(jobs), "Expected 2 jobs in list output")

	expectedJobIDs := jobIDs[:2]

	// Check that the job IDs match the expected values
	for i, job := range jobs {
		s.Require().Equal(expectedJobIDs[i], job.ID, "Job ID does not match expected")
	}

	// Verify that the message about fetching more records is present
	expectedMessage := fmt.Sprintf("To fetch more records use:\n\tbacalhau job list --limit %d --next-token", 2)
	s.Require().Contains(remainingOutput, expectedMessage, "Pagination message not found in output")
}

// ExtractJSONOutput extracts JSON data from the output
func ExtractJSONOutput(output string) (jsonData string, remainingOutput string, err error) {
	start := strings.Index(output, "[")
	if start == -1 {
		return "", "", fmt.Errorf("JSON data not found in output")
	}
	end := strings.LastIndex(output, "]")
	if end == -1 || end < start {
		return "", "", fmt.Errorf("JSON data not properly terminated in output")
	}
	jsonData = output[start : end+1]
	remainingOutput = output[end+1:]
	return jsonData, remainingOutput, nil
}
