package bacalhau

import (
	"fmt"
	"testing"

	_ "github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/stretchr/testify/assert"
)

func TestSubmitSyntaxErrors(t *testing.T) {
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
		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			err := system.CheckBashSyntax(tc.cmds)

			if tc.error_code != 0 {
				error_content := err.Error()
				assert.Error(t, err, fmt.Sprintf("Error was expected, but none found: %s", tc.expected_error_contains))
				assert.Contains(t, error_content, tc.expected_error_contains, fmt.Sprintf("Error was expected to contain: %s", tc.expected_error_contains))
			} else {
				assert.NoError(t, err, "Error in running command.")
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

f1`  // Unmatched bracket
)
