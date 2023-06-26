//go:build unit || !integration

package cancel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/testdata"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type CancelSuite struct {
	cmdtesting.BaseSuite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestCancelSuite(t *testing.T) {
	docker.MustHaveDocker(t)

	suite.Run(t, new(CancelSuite))
}

func (suite *CancelSuite) TestCancelTerminalJob() {
	ctx := context.Background()
	_, stdout, err := cmdtesting.ExecuteTestCobraCommandWithStdinBytes(testdata.TaskDockerJson.Data, "create",
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
	)
	require.NoError(suite.T(), err, "Error submitting job")

	job := testutils.GetJobFromTestOutput(ctx, suite.T(), suite.Client, stdout)
	suite.T().Logf("Created job %s", job.Metadata.ID)

	_, stdout, err = cmdtesting.ExecuteTestCobraCommand("cancel",
		job.Metadata.ID,
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
	)
	require.ErrorContains(suite.T(), err, "already in a terminal state")
}

func (suite *CancelSuite) TestCancelJob() {
	ctx := context.Background()

	_, stdout, err := cmdtesting.ExecuteTestCobraCommandWithStdinBytes(testdata.JsonJobCancel.Data, "create",
		"--wait=false",
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
	)
	require.NoError(suite.T(), err, "Error submitting job")

	// Read the job ID from stdout of the create command and make sure
	// we remove any whitespace before passing it to the Get call.
	stdout = strings.TrimSpace(stdout)

	jobInfo, _, err := suite.Client.Get(ctx, stdout)
	require.NoError(suite.T(), err, "Error finding newly created job")

	_, stdout, err = cmdtesting.ExecuteTestCobraCommand("cancel",
		jobInfo.Job.Metadata.ID,
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
	)
	require.NoError(suite.T(), err, "Error cancelling job")

	successMsg := fmt.Sprintf("Job successfully canceled. Job ID: %s", jobInfo.Job.Metadata.ID)
	require.Contains(suite.T(), stdout, successMsg)
}
