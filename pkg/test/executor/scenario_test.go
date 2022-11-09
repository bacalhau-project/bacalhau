package executor

import (
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
)

func TestNonDockerScenarios(t *testing.T) {
	for _, testCase := range scenario.GetAllScenarios() {
		if testCase.GetJobSpec().Engine == model.EngineDocker {
			continue
		}

		for _, storageDriverFactory := range scenario.StorageDriverFactories {
			t.Run(
				strings.Join([]string{testCase.Name, storageDriverFactory.Name}, "-"),
				func(t *testing.T) { RunTestCase(t, testCase) },
			)
		}
	}
}
