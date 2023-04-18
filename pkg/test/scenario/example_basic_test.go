package scenario

import (
	"testing"

	"github.com/stretchr/testify/suite"

	wasm_spec "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var basicScenario Scenario

func init() {
	engineSpec := (&wasm_spec.JobSpecWasm{
		EntryPoint: "_start",
	}).AsEngineSpec()
	basicScenario = Scenario{
		Inputs: ManyStores(
			StoredText("hello, world!", "/inputs"),
			StoredFile("../../../testdata/wasm/cat/main.wasm", "/job"),
		),
		Outputs: []model.StorageSpec{},
		Spec: model.Spec{
			EngineSpec: engineSpec,
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
	suite.RunScenario(basicScenario)
}
