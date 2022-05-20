package devstack

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
)

func TestCatFileStdout(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.CatFileToStdout(t),
		3,
	)
}

func TestCatFileOutputVolume(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.CatFileToVolume(t),
		3,
	)
}

func TestGrepFile(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.GrepFile(t),
		3,
	)
}

func TestSedFile(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.SedFile(t),
		3,
	)
}

func TestAwkFile(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.AwkFile(t),
		3,
	)
}
