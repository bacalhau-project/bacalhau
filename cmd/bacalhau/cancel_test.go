//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/docker"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type CancelSuite struct {
	BaseSuite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestCancelSuite(t *testing.T) {
	docker.MustHaveDocker(t)

	suite.Run(t, new(CancelSuite))
}

func (suite *CancelSuite) TestCancelTerminalJob() {
	testFile := "../../pkg/model/tasks/docker_task.json"

	ctx := context.Background()
	_, stdout, err := ExecuteTestCobraCommand(suite.T(), "create",
		"--api-host", suite.host,
		"--api-port", suite.port,
		testFile,
	)
	require.NoError(suite.T(), err, "Error submitting job")

	job := testutils.GetJobFromTestOutput(ctx, suite.T(), suite.client, stdout)
	suite.T().Logf("Created job %s", job.Metadata.ID)

	_, stdout, err = ExecuteTestCobraCommand(suite.T(), "cancel",
		job.Metadata.ID,
		"--api-host", suite.host,
		"--api-port", suite.port,
	)
	require.NoError(suite.T(), err, "Error cancelling job")
	require.Contains(suite.T(), stdout, "already in a terminal state")
}

func (suite *CancelSuite) TestCancelJob() {
	testFile := "../../testdata/job_cancel.json"

	ctx := context.Background()

	_, stdout, err := ExecuteTestCobraCommand(suite.T(), "create",
		"--wait=false",
		"--api-host", suite.host,
		"--api-port", suite.port,
		testFile,
	)
	require.NoError(suite.T(), err, "Error submitting job")

	// Read the job ID from stdout of the create command and make sure
	// we remove any whitespace before passing it to the Get call.
	stdout = strings.TrimSpace(stdout)

	jobInfo, _, err := suite.client.Get(ctx, stdout)
	require.NoError(suite.T(), err, "Error finding newly created job")

	_, stdout, err = ExecuteTestCobraCommand(suite.T(), "cancel",
		jobInfo.Job.Metadata.ID,
		"--api-host", suite.host,
		"--api-port", suite.port,
	)
	require.NoError(suite.T(), err, "Error cancelling job")

	successMsg := fmt.Sprintf("Job successfully canceled. Job ID: %s", jobInfo.Job.Metadata.ID)
	require.Contains(suite.T(), stdout, successMsg)
}
