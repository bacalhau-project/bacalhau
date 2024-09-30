//go:build integration || !unit

package devstack

import (
	"fmt"
	"os"
	"testing"

	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"

	"github.com/stretchr/testify/suite"
)

type DefaultPublisherSuite struct {
	scenario.ScenarioRunner
}

func TestDefaultPublisherSuite(t *testing.T) {
	suite.Run(t, new(DefaultPublisherSuite))
}

func (s *DefaultPublisherSuite) TestNoDefaultPublisher() {
	testcase := scenario.Scenario{
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					Engine: wasmmodels.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
						WithEntrypoint("_start").
						WithParameters(
							"data/hello.txt",
							"does/not/exist.txt",
						).
						MustBuild(),
				},
			},
		},
		ResultsChecker: expectResultsNone,
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}

func (s *DefaultPublisherSuite) TestDefaultPublisher() {
	stack := &scenario.StackConfig{}
	localSpecConfig := local.NewSpecConfig()
	stack.JobDefaults.Batch.Task.Publisher.Type = localSpecConfig.Type
	stack.JobDefaults.Batch.Task.Publisher.Params = localSpecConfig.Params
	testcase := scenario.Scenario{
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					Engine: wasmmodels.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
						WithEntrypoint("_start").
						WithParameters(
							"data/hello.txt",
							"does/not/exist.txt",
						).
						MustBuild(),
				},
			},
		},
		Stack:          stack,
		ResultsChecker: expectResultsSome,
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}

func expectResultsNone(resultsDir string) error {
	fcount := fileCount(resultsDir)
	if fcount == 0 {
		return nil
	}

	return fmt.Errorf("expected no files in %s, found %d", resultsDir, fcount)
}

func expectResultsSome(resultsDir string) error {
	fcount := fileCount(resultsDir)
	if fcount > 0 {
		return nil
	}

	return fmt.Errorf("expected some files in %s, found %d", resultsDir, fcount)
}

func fileCount(directory string) int {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0
	}
	return len(entries)
}
