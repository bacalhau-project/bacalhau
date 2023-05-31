package scenario

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"

	"github.com/stretchr/testify/suite"
)

func basicScenario(t testing.TB) Scenario {
	return Scenario{
		Inputs: ManyStores(
			StoredText("hello, world!", "/inputs"),
			StoredFile("../../../testdata/wasm/cat/main.wasm", "/job"),
		),
		Outputs: []spec.Storage{},
		Spec: model.Spec{
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
			),
		},
		ResultsChecker: FileEquals(model.DownloadFilenameStdout, "hello, world!\n"),
		JobCheckers:    WaitUntilSuccessful(1),
	}
}

type ExampleTest struct {
	ScenarioRunner
}

func Example_basic() {
	// In a real example, use the testing.T passed to the TestXxx method.
	suite.Run(&testing.T{}, new(ExampleTest))
}

func (suite *ExampleTest) TestRun() {
	suite.RunScenario(basicScenario(suite.T()))
}
