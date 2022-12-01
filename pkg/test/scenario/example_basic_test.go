package scenario

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/suite"
)

var basicScenario Scenario = Scenario{
	Inputs:   StoredText("hello, world!", "/inputs"),
	Contexts: StoredFile("../../../testdata/wasm/cat/main.wasm", "/job"),
	Outputs:  []model.StorageSpec{},
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint: "_start",
		},
	},
	ResultsChecker: FileEquals(ipfs.DownloadFilenameStdout, "hello, world!\n"),
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
