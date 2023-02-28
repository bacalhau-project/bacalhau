//go:build unit || !integration

package bacalhau

import (
	"fmt"
	"testing"

	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ValidateSuite struct {
	BaseSuite
}

func TestValidateSuite(t *testing.T) {
	suite.Run(t, new(ValidateSuite))
}

func (s *ValidateSuite) TestValidate() {

	tests := map[string]struct {
		testFile string
		valid    bool
	}{
		"validJobFile":   {testFile: "../../testdata/job-noop.yaml", valid: true},
		"InvalidJobFile": {testFile: "../../testdata/job-noop-invalid.yml", valid: false},
	}
	for name, test := range tests {
		func() {
			Fatal = FakeFatalErrorHandler

			_, out, err := ExecuteTestCobraCommand("validate",
				"--api-host", s.host,
				"--api-port", s.port,
				test.testFile,
			)

			require.NoError(s.T(), err)

			// fmt.Print(s)
			if test.valid {
				require.Contains(s.T(), out, "The Job is valid", fmt.Sprintf("%s: Jobspec Invalid", name))
			} else {
				fatalError, err := testutils.FirstFatalError(s.T(), out)
				require.NoError(s.T(), err)
				require.Contains(s.T(), fatalError.Message, "The Job is not valid.", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
				require.Contains(s.T(), fatalError.Message, "APIVersion is required", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
			}
		}()

	}
}
