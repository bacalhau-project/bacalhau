//go:build unit || !integration

package templates

import (
	"runtime"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/util/stringutils"

	"github.com/stretchr/testify/assert"
)

func TestLongDesc(t *testing.T) {
	actual := LongDesc(`
		Create a job from a file or from stdin.

		JSON and YAML formats are accepted.
	`)

	actual = stringutils.CrossPlatformNormalizeLineEndings(actual)
	want := `Create a job from a file or from stdin.

 JSON and YAML formats are accepted.`
	want = stringutils.CrossPlatformNormalizeLineEndings(want)

	assert.Equal(t, want, actual)
}

func TestExamples(t *testing.T) {
	actual := Examples(`
		# Describe a job with the full ID
		bacalhau describe j-e3f8c209-d683-4a41-b840-f09b88d087b9

		# Describe a job with the a shortened ID
		bacalhau describe j-47805f5c

		# Describe a job and include all server and local events
		bacalhau describe --include-events j-b6ad164a
`)

	actual = stringutils.CrossPlatformNormalizeLineEndings(actual)

	want := `  # Describe a job with the full ID
  bacalhau describe j-e3f8c209-d683-4a41-b840-f09b88d087b9

  # Describe a job with the a shortened ID
  bacalhau describe j-47805f5c

  # Describe a job and include all server and local events
  bacalhau describe --include-events j-b6ad164a`

	want = stringutils.CrossPlatformNormalizeLineEndings(want)

	assert.Equal(t, want, actual, "Examples did not match - GOOS: %s\nGOARCH: %s", runtime.GOOS, runtime.GOARCH)
}
