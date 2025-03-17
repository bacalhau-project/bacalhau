package testutils

import (
	"runtime"
	"testing"
)

// SkipIfNotLinux skips the test if not running on Linux.
// issueURL is optional and can be used to reference a GitHub issue tracking platform support.
func SkipIfNotLinux(t *testing.T, issueURL string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("Test only supported on Linux", issueURL)
	}
}
