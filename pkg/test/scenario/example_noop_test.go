package scenario

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var noopScenario Scenario = Scenario{
	Stack: &StackConfig{
		ExecutorConfig: noop.ExecutorConfig{
			ExternalHooks: noop.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, jobID string, resultsDir string) (*model.RunCommandResult, error) {
					return executor.WriteJobResults(resultsDir, strings.NewReader("hello, world!\n"), nil, 0, nil)
				},
			},
		},
	},
	Spec: model.Spec{
		Engine: model.EngineNoop,
		Wasm: model.JobSpecWasm{
			EntryPoint: "_start",
		},
	},
	ResultsChecker: FileEquals(model.DownloadFilenameStdout, "hello, world!\n"),
	JobCheckers:    WaitUntilSuccessful(1),
}

type NoopTest struct {
	ScenarioRunner
}

func Example_noop() {
	// In a real example, use the testing.T passed to the TestXxx method.
	suite.Run(&testing.T{}, new(NoopTest))
}

func (suite *NoopTest) TestRunNoop() {
	suite.RunScenario(noopScenario)
}
