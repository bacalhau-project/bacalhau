//go:build unit || !integration

package validate_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util/handler"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/testdata"
)

type ValidateSuite struct {
	cmdtesting.BaseSuite
}

func TestValidateSuite(t *testing.T) {
	suite.Run(t, new(ValidateSuite))
}

func (s *ValidateSuite) TestValidate() {
	tests := map[string]struct {
		testFile *testdata.Fixture
		valid    bool
	}{
		"validJobFile":   {testFile: testdata.YamlJobNoop, valid: true},
		"InvalidJobFile": {testFile: testdata.YamlJobNoopInvalid, valid: false},
	}
	for name, test := range tests {
		s.Run(name, func() {
			handler.Fatal = handler.FakeFatalErrorHandler

			_, out, err := cmdtesting.ExecuteTestCobraCommand("validate",
				"--api-host", s.Host,
				"--api-port", fmt.Sprint(s.Port),
				test.testFile.AsTempFile(s.T(), fmt.Sprintf("%s.*.yaml", name), s.T().TempDir()),
			)
			require.NoError(s.T(), err)

			if test.valid {
				require.Contains(s.T(), out, "The Job is valid", fmt.Sprintf("%s: Jobspec Invalid", name))
			} else {
				fatalError, err := testutils.FirstFatalError(s.T(), out)
				require.NoError(s.T(), err)
				require.Contains(s.T(), fatalError.Message, "The Job is not valid.", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
				require.Contains(s.T(), fatalError.Message, "APIVersion is required", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
			}
		})

	}
}
