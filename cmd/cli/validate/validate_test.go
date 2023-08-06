//go:build unit || !integration

package validate_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util"
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
			util.Fatal = util.FakeFatalErrorHandler

			_, out, err := cmdtesting.ExecuteTestCobraCommand("validate",
				"--api-host", s.Host,
				"--api-port", fmt.Sprint(s.Port),
				test.testFile.AsTempFile(s.T(), fmt.Sprintf("%s.*.yaml", name)),
			)
			s.Require().NoError(err)

			if test.valid {
				s.Require().Contains(out, "The Job is valid", fmt.Sprintf("%s: Jobspec Invalid", name))
			} else {
				fatalError, err := testutils.FirstFatalError(s.T(), out)
				s.Require().NoError(err)
				s.Require().Contains(fatalError.Message, "The Job is not valid.", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
				s.Require().Contains(fatalError.Message, "APIVersion is required", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
			}
		})

	}
}
