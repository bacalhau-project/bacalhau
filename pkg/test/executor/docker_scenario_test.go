//go:build !(unit && (windows || darwin))

package executor

import (
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
)

func TestDockerScenarios(t *testing.T) {
	for _, testCase := range scenario.GetAllScenarios() {
		if testCase.Spec.Engine != model.EngineDocker {
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
