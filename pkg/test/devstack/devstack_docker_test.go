package devstack

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	dockertests "github.com/filecoin-project/bacalhau/pkg/test/docker"
)

func TestDevStackDockerStorage(t *testing.T) {

	tests := dockertests.GetTestCases(t)

	for i, test := range tests {

		if i > 0 {
			continue
		}

		devStackDockerStorageTest(
			t,
			test.Name,
			test.SetupStorage,
			test.ResultsChecker,
			test.GetJobSpec,
			3,
		)

	}
}
