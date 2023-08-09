//go:build integration || !unit

package executor

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

func TestScenarios(t *testing.T) {
	for name, testCase := range scenario.GetAllScenarios(t) {
		t.Run(
			name,
			func(t *testing.T) {
				t.Log(name)
				docker.EngineSpecRequiresDocker(t, testCase.Spec.EngineSpec)
				RunTestCase(t, testCase)
			},
		)
	}
}
