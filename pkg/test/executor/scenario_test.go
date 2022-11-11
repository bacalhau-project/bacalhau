//go:build !integration

package executor

import (
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
)

func TestScenarios(t *testing.T) {
	for _, testCase := range scenario.GetAllScenarios() {
		for _, storageDriverFactory := range scenario.StorageDriverFactories {
			t.Run(
				strings.Join([]string{testCase.Name, storageDriverFactory.Name}, "-"),
				func(t *testing.T) {
					testutils.MaybeNeedDocker(t, testCase.Spec.Engine == model.EngineDocker)
					RunTestCase(t, testCase)
				},
			)
		}
	}
}
