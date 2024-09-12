//go:build unit || !integration

package job_test

import (
	"context"
	"fmt"
	"testing"

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
			_, out, err := s.ExecuteTestCobraCommandWithStdinBytes(tc.Data, "job", "run")

			fmt.Println(tc.Data)

			if tc.Invalid {
				require.Error(s.T(), err, "Should have seen error submitting job")
			} else {
				require.NoError(s.T(), err, "Error submitting job")
				testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
			}
		})
	}
}
