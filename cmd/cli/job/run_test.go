//go:build unit || !integration

package job_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/testdata"
)

type RunSuite struct {
	cmdtesting.BaseSuite
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(RunSuite))
}

func (s *RunSuite) TestRun() {
	for _, tc := range testdata.AllFixtures() {
		s.Run(tc.Description, func() {
			if tc.RequiresS3() && !s3helper.CanRunS3Test() {
				// Skip the S3 tests if we have no AWS credentials installed
				s.T().Skip("No valid AWS credentials found")
			}

			if tc.RequiresDocker() {
				docker.MustHaveDocker(s.T())
			}

			ctx := context.Background()
			_, out, err := cmdtesting.ExecuteTestCobraCommandWithStdinBytes(tc.Data, "job", "run",
				"--api-host", s.Host,
				"--api-port", fmt.Sprint(s.Port),
			)

			fmt.Println(tc.Data)

			require.NoError(s.T(), err, "Error submitting job")
			if tc.Invalid {
				assert.Contains(s.T(), out, userstrings.JobSpecBad)
			} else {
				testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
			}
		})
	}
}
