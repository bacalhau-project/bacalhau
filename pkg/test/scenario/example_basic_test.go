package scenario

import (
	"testing"

	"github.com/stretchr/testify/suite"

	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

func basicScenario(t testing.TB) Scenario {
	return Scenario{
		Inputs: ManyStores(
			StoredText("hello, world!", "/inputs"),
			StoredFile("../../../testdata/wasm/cat/main.wasm", "/job"),
		),
		Outputs:        []model.StorageSpec{},
		ResultsChecker: FileEquals(model.DownloadFilenameStdout, "hello, world!\n"),
		JobCheckers:    WaitUntilSuccessful(1),
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				// TODO(forrest): [correctness] this isn't a valid wasm engine spec - it needs an entry module
				// but leaving as is to preserve whatever behaviour this test is after.
				model.NewWasmEngineBuilder(model.StorageSpec{}).
					WithEntrypoint("_start").
					Build(),
			),
		),
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
