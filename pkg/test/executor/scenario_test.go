//go:build integration || !unit

package executor

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	docker2 "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

func TestScenarios(t *testing.T) {
	for name, testCase := range scenario.GetAllScenarios(t) {
		t.Run(
			name,
			func(t *testing.T) {
				docker.MaybeNeedDocker(t, testCase.Spec.Engine.Schema == docker2.EngineSchema.Cid())
				RunTestCase(t, testCase)
			},
		)
	}
}
