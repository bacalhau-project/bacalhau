package docker

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// MustHaveDocker will skip the test if the test is running in an environment that cannot support cross-platform
// Docker images.
func MustHaveDocker(t testing.TB) {
	MaybeNeedDocker(t, true)
}

// EngineSpecRequiresDocker will skip the test if the test is running in an environment that cannot support cross-platform
// Docker images, and the passed model.EngineSpec type is model.EngineDocker
func EngineSpecRequiresDocker(t testing.TB, engineSpec model.EngineSpec) {
	MaybeNeedDocker(t, engineSpec.Engine() == model.EngineDocker)
}

// MaybeNeedDocker will skip the test if the test is running in an environment that cannot support cross-platform
// Docker images, and the passed boolean flag is true.
func MaybeNeedDocker(t testing.TB, needDocker bool) {
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
