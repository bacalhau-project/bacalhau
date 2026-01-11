package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.ptx.dk/multierrgroup"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// A CheckResults is a function that will examine job output that has been
// written to storage and assert something about it. If the condition it is
// checking is false, it returns an error, else it returns nil.
type CheckResults func(resultsDir string) error

// A CheckCommandResults is a function that will examine the results of an executed command
type CheckCommandResults func(result *models.RunCommandResult) error

// FileContains returns a CheckResults that asserts that the expected string is
// in the output file and that the file itself is of the correct size. If
// expectedLine is set to -1 then a line-check is not performed.
func FileContains(
	outputFilePath string,
	expectedStrings []string,
	expectedLines int,
) CheckResults {
	return func(resultsDir string) error {
		if err := FileExists(outputFilePath)(resultsDir); err != nil {
			return err
		}
		outputFile := filepath.Join(resultsDir, outputFilePath)
		resultsContent, err := os.ReadFile(outputFile) //nolint:gosec // G304: outputFile from test fixture, controlled
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
		if err := FileExists(outputFilePath)(resultsDir); err != nil {
			return err
		}
		outputFile := filepath.Join(resultsDir, outputFilePath)
		resultsContent, err := os.ReadFile(outputFile) //nolint:gosec // G304: outputFile from test fixture, controlled
		if err != nil {
			return err
		}

		if string(resultsContent) != expectedString {
			return fmt.Errorf("%s: content mismatch:\nExpected: %q\nActual: %q", outputFile, expectedString, resultsContent)
		}
		return nil
	}
}

// FileExists returns a CheckResults that asserts the file exists in the results directory
func FileExists(outputFilePath string) CheckResults {
	return func(resultsDir string) error {
		outputFile := filepath.Join(resultsDir, outputFilePath)
		_, err := os.Stat(outputFile)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s\nFiles found in results directory:\n%s",
					outputFile, listFiles(resultsDir))
			}
			return err
		}
		return nil
	}
}

// FileNotExists returns a CheckResults that asserts the file does not exist in the results directory
func FileNotExists(outputFilePath string) CheckResults {
	return func(resultsDir string) error {
		outputFile := filepath.Join(resultsDir, outputFilePath)
		_, err := os.Stat(outputFile)
		if err == nil {
			return fmt.Errorf("file exists but should not: %s\nFiles found in results directory:\n%s",
				outputFile, listFiles(resultsDir))
		}
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
}

func listFiles(dir string) string {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relPath != "." {
			files = append(files, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Sprintf("error listing files: %v", err)
	}
	if len(files) == 0 {
		return "no files found"
	}
	return strings.Join(files, "\n")
}

// ManyChecks returns a CheckResults that runs the passed checkers and returns
// an error if any of them fail.
func ManyChecks(checks ...CheckResults) CheckResults {
	return func(resultsDir string) error {
		var wg multierrgroup.Group
		for _, check := range checks {
			wg.Go(func() error { return check(resultsDir) })
		}
		return wg.Wait()
	}
}

// ErrorMessageContains returns a CheckCommandResults that asserts that the expected string is
// in the error message of the command result
func ErrorMessageContains(expectedString string) CheckCommandResults {
	return func(result *models.RunCommandResult) error {
		if !strings.Contains(result.ErrorMsg, expectedString) {
			return fmt.Errorf("error message mismatch:\nExpected Contains: %q\nActual: %q", expectedString, result.ErrorMsg)
		}
		return nil
	}
}
