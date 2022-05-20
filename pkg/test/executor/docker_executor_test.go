package docker

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
)

func TestCatFileStdout(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.CatFileToStdout(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestCatFileOutputVolume(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.CatFileToVolume(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestGrepFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.GrepFile(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestSedFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.SedFile(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestAwkFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.AwkFile(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}
