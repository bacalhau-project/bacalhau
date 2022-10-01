package bacalhau

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
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

//before all the suite
func (s *ValidateSuite) SetupSuite() {

}

//before each test
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

			trimmedString := strings.TrimSpace(out)
			// fmt.Print(s)
			if test.valid {
				require.Equal(s.T(), "The Job is valid", trimmedString, fmt.Sprintf("%s: Jobspec Invalid", name))
			} else {
				require.Equal(s.T(), trimmedString[0:21], "The Job is not valid.", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
				require.Contains(s.T(), trimmedString, "APIVersion is required", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
			}
		}()

	}
}
