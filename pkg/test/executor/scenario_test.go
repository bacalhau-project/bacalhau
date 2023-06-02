//go:build integration || !unit

package executor

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	spec_docker "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

func TestScenarios(t *testing.T) {
	for name, testCase := range scenario.GetAllScenarios(t) {
		t.Run(
			name,
			func(t *testing.T) {
				docker.MaybeNeedDocker(t, testCase.Spec.Engine.Schema == spec_docker.EngineType)
				RunTestCase(t, testCase)
			},
		)
	}
}
