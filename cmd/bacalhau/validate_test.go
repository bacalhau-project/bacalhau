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
func (suite *ValidateSuite) SetupSuite() {

}

//before each test
func (suite *ValidateSuite) SetupTest() {
	require.NoError(suite.T(), system.InitConfigForTesting())
	suite.rootCmd = RootCmd
}

func (suite *ValidateSuite) TearDownTest() {

}

func (suite *ValidateSuite) TearDownAllSuite() {

}

func (suite *ValidateSuite) TestValidate() {

	tests := map[string]struct {
		testFile string
		valid    bool
	}{
		"validJobFile":   {testFile: "../../testdata/job.yaml", valid: true},
		"InvalidJobFile": {testFile: "../../testdata/job-invalid.yml", valid: false},
	}
	for name, test := range tests {
		func() {
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			*OV = *NewValidateOptions()

			parsedBasedURI, err := url.Parse(c.BaseURI)
			require.NoError(suite.T(), err)

			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "validate",
				"--api-host", host,
				"--api-port", port,
				test.testFile,
			)

			require.NoError(suite.T(), err)

			trimmedString := strings.TrimSpace(out)
			// fmt.Print(s)
			if test.valid {
				require.Equal(suite.T(), "The Job is valid", trimmedString, fmt.Sprintf("%s: Jobspec Invalid", name))
			} else {
				require.Equal(suite.T(), trimmedString[0:21], "The Job is not valid.", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
				require.Contains(suite.T(), trimmedString, "APIVersion is required", fmt.Sprintf("%s: Jobspec Invalid returning valid", name))
			}
		}()

	}
}
