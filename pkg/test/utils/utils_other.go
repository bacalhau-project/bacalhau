//go:build !arm64

package testutils

import (
	"testing"
)

func SkipIfArm(_ *testing.T, _ string) {
}
