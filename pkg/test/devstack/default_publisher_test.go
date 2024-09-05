//go:build integration || !unit

package devstack

import (
	"fmt"
	"os"
	"testing"

	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	s.T().Skip("This test is invalid see TODO")
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
		// TODO(unsure): inorder for the default publisher to apply, the task needs ResultPaths set with no publisher.
		// We can't results paths directly on the task because scenarios use this field to override tasks result paths.
		// So here we are setting the ResultsPath on the above task.
		// By omitting the publisher from the task we allow allow the orchestrator to populate it with job defaults,
		// which sets the default publisher.
		// However, in https://github.com/bacalhau-project/bacalhau/pull/2802/files#diff-7c0b9e1e39b2a2f5a00159ab3b4bed52d426dfa345498495ee8ee69dee6cc56aR130-R131
		// validation was added the marks tasks with result paths and no publisher as invalid, so this test can't run
		// and thus we skip it.
		// Conversation: https://github.com/bacalhau-project/bacalhau/pull/4333#discussion_r1745302676
		Outputs: []*models.ResultPath{
			{
				Name: "outputs",
				Path: "/outputs",
			},
		},

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
