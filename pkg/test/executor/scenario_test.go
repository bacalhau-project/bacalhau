//go:build integration

package executor

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
)

func TestScenarios(t *testing.T) {
	for name, testCase := range scenario.GetAllScenarios() {
		t.Run(
			name,
			func(t *testing.T) {
				docker.MaybeNeedDocker(t, testCase.Spec.Engine == model.EngineDocker)
				RunTestCase(t, testCase)
			},
		)
	}
}
