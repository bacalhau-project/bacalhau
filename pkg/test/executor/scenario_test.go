//go:build integration || !unit

package executor

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

func TestScenarios(t *testing.T) {
	for name, testCase := range scenario.GetAllScenarios() {
		t.Run(
			name,
			func(t *testing.T) {
				t.Log(name)
				docker.MaybeNeedDocker(t, testCase.Spec.Engine == model.EngineDocker)
				RunTestCase(t, testCase)
			},
		)
	}
}
