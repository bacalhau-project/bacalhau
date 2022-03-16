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
		"missing_quote":  {cmds: []string{MISSING_QUOTE}, error_code: 1, expected_output_contains: "Couldn't parse this double quoted string", expected_error_contains: ""},
		"unescaped_find": {cmds: []string{UNQUOTED_FIND_PATTERN}, error_code: 1, expected_output_contains: "Quote the parameter", expected_error_contains: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			err, stdout, stderr := system.CheckBashSyntax(tc.cmds)

			if tc.error_code != 0 {
			} else {
				assert.NoError(t, err, "Error in running command.")
			}
			assert.Contains(t, stdout.String(), tc.expected_output_contains, fmt.Sprintf("Expected output doesn't match: %s", stdout.String()))
			assert.Contains(t, stderr.String(), tc.expected_error_contains, fmt.Sprintf("Expected output doesn't match: %s", stderr.String()))

			// stack, cancelFunction := setupTest(t, tc.nodes, tc.badActors)
			// defer teardownTest(stack, cancelFunction)

			// c := make(chan os.Signal, 1)
			// signal.Notify(c, os.Interrupt)
			// go func() {
			// 	for range c {
			// 		teardownTest(stack, cancelFunction)
			// 		os.Exit(1)
			// 	}
			// }()

			// job, _, err := execute_command(t, stack, commands[0], "", tc.concurrency, tc.confidence, tc.tolerance)
			// assert.NoError(t, err, "Error executing command: %+v", err)

			// resultsList, err := system.ProcessJobIntoResults(job)
			// assert.NoError(t, err, "Error processing job into results: %+v", err)

			// correctGroup, incorrectGroup, err := traces.ProcessResults(job, resultsList)

			// assert.Equal(t, (len(correctGroup)-len(incorrectGroup)) == tc.nodes, fmt.Sprintf("Expected %d good actors, got %d", tc.nodes, len(correctGroup)))
			// assert.Equal(t, (len(incorrectGroup)) == tc.badActors, fmt.Sprintf("Expected %d bad actors, got %d", tc.badActors, len(incorrectGroup)))
			// assert.NoError(t, err, "Expected to run with no error. Actual: %+v", err)

		})
	}
}

// https://github.com/koalaman/shellcheck
var (
	GOOD_PYTHON           = `python3 -c "time.sleep(10); %s"`
	MISSING_QUOTE         = `python3 -c "time.sleep(10); %s` // note that trailing quote is missing
	UNQUOTED_FIND_PATTERN = `find . -name *.ogg`             // Unquoted find/grep patterns
)
