//go:build integration || !unit

package executor

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
)

func TestScenarios(t *testing.T) {
	for name, testCase := range scenario.GetAllScenarios() {
		t.Run(
			name,
			func(t *testing.T) {
				testutils.MaybeNeedDocker(t, testCase.Spec.Engine == model.EngineDocker)
				RunTestCase(t, testCase)
			},
		)
	}
}
