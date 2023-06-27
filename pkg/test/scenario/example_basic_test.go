package scenario

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var basicScenario Scenario = Scenario{
	Inputs: ManyStores(
		StoredText("hello, world!", "/inputs"),
		StoredFile("../../../testdata/wasm/cat/main.wasm", "/job"),
	),
	Outputs: []model.StorageSpec{},
	Spec: model.Spec{
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(model.StorageSpec{}, "_start", nil, nil, nil),
	},
	ResultsChecker: FileEquals(model.DownloadFilenameStdout, "hello, world!\n"),
	JobCheckers:    WaitUntilSuccessful(1),
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
