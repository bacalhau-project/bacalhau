package system

import (
	"fmt"
	"strings"
)

// Function for validating the workdir of a docker command.
func ValidateWorkingDir(jobWorkingDir string) error {
	if jobWorkingDir != "" {
		if !strings.HasPrefix(jobWorkingDir, "/") {
			// This mirrors the implementation at path/filepath/path_unix.go#L13 which
			// we reuse here to get cross-platform working dir detection. This is
			// necessary (rather than using IsAbs()) because clients may be running on
			// Windows/Plan9 but we want to check inside Docker (linux).
			return fmt.Errorf("workdir must be an absolute path. Passed in: %s", jobWorkingDir)
		}
	}
	return nil
}
