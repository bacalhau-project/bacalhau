//go:build unit || !integration

package validate_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
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
		"validJobFile":   {testFile: testdata.DockerJobYAML, valid: true},
		"InvalidJobFile": {testFile: testdata.DockerJobYAMLInvalid, valid: false},
	}

	for name, test := range tests {
		s.Run(name, func() {
			tempFile := test.testFile.AsTempFile(s.T(), fmt.Sprintf("%s.*.yaml", name))
			fmt.Println("Testing with file:", tempFile) // Debug: Print the actual temp file path

			_, out, err := s.ExecuteTestCobraCommand("validate", "-f", tempFile)

			if test.valid {
				require.NoError(s.T(), err, "Expected no error for valid input")
				require.Contains(s.T(), out, "The jobspec is valid", fmt.Sprintf("%s: Jobspec should be valid", name))
			} else {
				require.Error(s.T(), err, fmt.Sprintf("%s: Expected an error for invalid input", name))
				require.Contains(s.T(), out, "Error:", fmt.Sprintf("%s: Expected validation errors", name))
			}
		})
	}
}
