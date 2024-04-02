package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.ptx.dk/multierrgroup"
)

// A CheckResults is a function that will examine job output that has been
// written to storage and assert something about it. If the condition it is
// checking is false, it returns an error, else it returns nil.
type CheckResults func(resultsDir string) error

// FileContains returns a CheckResults that asserts that the expected string is
// in the output file and that the file itself is of the correct size. If
// expectedLine is set to -1 then a line-check is not performed.
func FileContains(
	outputFilePath string,
	expectedStrings []string,
	expectedLines int,
) CheckResults {
	return func(resultsDir string) error {
		outputFile := filepath.Join(resultsDir, outputFilePath)
		resultsContent, err := os.ReadFile(outputFile)
		if err != nil {
			return err
		}

		actualLineCount := len(strings.Split(string(resultsContent), "\n"))
		if expectedLines != -1 && actualLineCount != expectedLines {
			return fmt.Errorf("%s: count mismatch:\nExpected: %d\nActual: %d", outputFile, expectedLines, actualLineCount)
		}

		for _, expectedString := range expectedStrings {
			if !strings.Contains(string(resultsContent), expectedString) {
				return fmt.Errorf("%s: content mismatch:\nExpected Contains: %q\nActual: %q", outputFile, expectedString, resultsContent)
			}
		}

		return nil
	}
}

// FileEquals returns a CheckResults that asserts that the expected string is
// exactly equal to the full contents of the output file.
func FileEquals(
	outputFilePath string,
	expectedString string,
) CheckResults {
	return func(resultsDir string) error {
		outputFile := filepath.Join(resultsDir, outputFilePath)
		resultsContent, err := os.ReadFile(outputFile)
		if err != nil {
			return err
		}

		if string(resultsContent) != expectedString {
			return fmt.Errorf("%s: content mismatch:\nExpected: %q\nActual: %q", outputFile, expectedString, resultsContent)
		}
		return nil
	}
}

// ManyCheckes returns a CheckResults that runs the passed checkers and returns
// an error if any of them fail.
func ManyChecks(checks ...CheckResults) CheckResults {
	return func(resultsDir string) error {
		var wg multierrgroup.Group
		for _, check := range checks {
			check := check
			wg.Go(func() error { return check(resultsDir) })
		}
		return wg.Wait()
	}
}
