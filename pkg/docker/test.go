package docker

import (
	"context"
	"os"
	"runtime"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

// MustHaveDocker will skip the test if the test is running in an environment that cannot support cross-platform
// Docker images.
func MustHaveDocker(t testingT) {
	MaybeNeedDocker(t, true)
}

// MaybeNeedDocker will skip the test if the test is running in an environment that cannot support cross-platform
// Docker images, and the passed boolean flag is true.
func MaybeNeedDocker(t testingT, needDocker bool) {
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

type testingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
	Fatalf(format string, args ...any)
	Skip(args ...any)
}
