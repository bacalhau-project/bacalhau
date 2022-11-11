//go:build !(unit && (windows || darwin))

package executor

import (
	"testing"
)

func TestDockerScenarios(t *testing.T) {
	// for _, testCase := range scenario.GetAllScenarios() {
	// 	if testCase.GetJobSpec().Engine != model.EngineDocker {
	// 		continue
	// 	}

	// 	for _, storageDriverFactory := range scenario.StorageDriverFactories {
	// 		t.Run(
	// 			strings.Join([]string{testCase.Name, storageDriverFactory.Name}, "-"),
	// 			func(t *testing.T) { RunTestCase(t, testCase) },
	// 		)
	// 	}
	// }
}
