package system

import (
	"fmt"
	"testing"

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

// Before all suite
func (suite *SystemScriptCheckerSuite) SetupAllSuite() {

}

// Before each test
func (suite *SystemScriptCheckerSuite) SetupTest() {
	require.NoError(suite.T(), InitConfigForTesting())
}

func (suite *SystemScriptCheckerSuite) TearDownTest() {
}

func (suite *SystemScriptCheckerSuite) TearDownAllSuite() {

}
func (suite *SystemScriptCheckerSuite) TestSubmitSyntaxErrors() {
	tests := map[string]struct {
		cmds                     []string
		error_code               int
		expected_output_contains string
		expected_error_contains  string
	}{
		"good_bash":      {cmds: []string{GOOD_PYTHON}, error_code: 0, expected_output_contains: "", expected_error_contains: ""},
		"missing_quote":  {cmds: []string{MISSING_QUOTE}, error_code: 1, expected_output_contains: "", expected_error_contains: "reached EOF without closing quote"},
		"unescaped_find": {cmds: []string{UNMATCHED_BRACKET}, error_code: 1, expected_output_contains: "", expected_error_contains: "reached EOF without matching"},
	}

	for name, tc := range tests {
		suite.T().Run(name, func(t *testing.T) {
			// t.Parallel()

			err := CheckBashSyntax(tc.cmds)

			if tc.error_code != 0 {
				error_content := err.Error()
				require.Error(t, err, fmt.Sprintf("Error was expected, but none found: %s", tc.expected_error_contains))
				require.Contains(t, error_content, tc.expected_error_contains, fmt.Sprintf("Error was expected to contain: %s", tc.expected_error_contains))
			} else {
				require.NoError(t, err, "Error in running command.")
			}

		})
	}
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

// https://github.com/koalaman/shellcheck
var (
	GOOD_PYTHON       = `python3 -c "time.sleep(10); %s"`
	MISSING_QUOTE     = `python3 -c "time.sleep(10); %s` // note that trailing quote is missing
	UNMATCHED_BRACKET = `function f1() {
    echo "Hello World"

f1`  // Unmatched bracket across several lines
	// TODO: Need to do a test for binary listed not on path (possible??)
)
