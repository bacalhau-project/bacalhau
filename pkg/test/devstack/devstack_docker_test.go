package devstack

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	dockertests "github.com/filecoin-project/bacalhau/pkg/test/docker"
)

func TestDevStackDockerStorage(t *testing.T) {

	tests := dockertests.GetTestCases(t)

	for _, test := range tests {

		DevStackDockerStorageTest(
			t,
			test.Name,
			test.SetupStorage,
			test.ResultsChecker,
			test.GetJobSpec,
			3,
		)

	}
}
