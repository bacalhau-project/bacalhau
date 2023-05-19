//go:build arm64

package testutils

import (
	"testing"
)

func SkipIfArm(t *testing.T, issueURL string) {
	t.Skip("Test does not pass natively on arm64", issueURL)
}
