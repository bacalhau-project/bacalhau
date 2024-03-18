package string

import (
	"runtime"
	"strings"
)

// Function that normalizes line endings to platform being run on.
// Useful for tests, but possibly useful elsewhere?
func CrossPlatformNormalizeLineEndings(s string) string {
	return crossPlatformNormalizeLineEndings(s, runtime.GOOS)
}

// Internal only function to allow injecting the platform for testing
func crossPlatformNormalizeLineEndings(s string, platform string) string {
	// Detect the platform
	lineEnding := "\n"
	if platform == "windows" {
		lineEnding = "\r\n"
	}

	// Use go's built-in splitter to split the string into lines
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")

	// Trim all whitespace from empty lines
	for i, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			lines[i] = ""
		}
	}

	// Now recombine the lines with the correct line ending
	s = strings.Join(lines, lineEnding)

	return s
}
