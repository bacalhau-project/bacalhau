//go:build !integration

package system

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SystemScriptCheckerSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSystemScriptCheckerSuite(t *testing.T) {
	suite.Run(t, new(SystemScriptCheckerSuite))
}

// Before each test
func (suite *SystemScriptCheckerSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	require.NoError(suite.T(), InitConfigForTesting())
}

func (suite *SystemScriptCheckerSuite) TestValidateWorkingDir() {
	tests := map[string]struct {
		path       string
		error_code int
	}{
		"good_path":       {path: "/", error_code: 0},
		"good_path_full":  {path: "/project/", error_code: 0},
		"relative_path":   {path: "../foo", error_code: 1},
		"relative_path_2": {path: "./foo", error_code: 1},
	}

	for name, tc := range tests {
		suite.T().Run(name, func(t *testing.T) {
			// t.Parallel()

			err := ValidateWorkingDir(tc.path)

			if tc.error_code != 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err, "Error in running command.")
			}
		})
	}
}
