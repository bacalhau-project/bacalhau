package bacalhau

import (
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ValidateSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

func TestValidateSuite(t *testing.T) {
	suite.Run(t, new(ValidateSuite))
}

// before all the suite
func (s *ValidateSuite) SetupSuite() {

}

// before each test
func (s *ValidateSuite) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting())
	s.rootCmd = RootCmd
}

func (s *ValidateSuite) TearDownTest() {

}

func (s *ValidateSuite) TearDownAllSuite() {

}

func (s *ValidateSuite) TestValidate() {

	tests := map[string]struct {
		testFile string
		valid    bool
	}{
		"validJobFile":   {testFile: "../../testdata/job.yaml", valid: true},
		"InvalidJobFile": {testFile: "../../testdata/job-invalid.yml", valid: false},
	}
	for name, test := range tests {
		func() {
			Fatal = FakeFatalErrorHandler

			c, cm := publicapi.SetupTests(s.T())
			defer cm.Cleanup()

			*OV = *NewValidateOptions()

			parsedBasedURI, err := url.Parse(c.BaseURI)
			require.NoError(s.T(), err)

			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "validate",
				"--api-host", host,
				"--api-port", port,
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
