package docker

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

// If the test is running in an environment that cannot support cross-platform
// Docker images, the test is skipped.
func MustHaveDocker(t *testing.T) {
	MaybeNeedDocker(t, true)
}

// If the test is running in an environment that cannot support cross-platform
// Docker images, and the passed boolean flag is true, the test is skipped.
func MaybeNeedDocker(t *testing.T, needDocker bool) {
	_, isCI := os.LookupEnv("CI")
	if needDocker && isCI && (runtime.GOOS == "windows" || runtime.GOOS == "darwin") {
		t.Skip("Cannot run this test in a", runtime.GOOS, "runtime on a CI environment because it requires Docker")
	}

	if needDocker {
		c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		require.NoError(t, err)

		_, err = c.Info(context.Background())
		if err != nil {
			t.Fatalf("Docker is not running")
		}
	}
}
