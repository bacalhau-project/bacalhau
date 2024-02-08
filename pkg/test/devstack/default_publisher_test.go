//go:build integration || !unit

package devstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
		Spec: testutils.MakeSpecWithOpts(s.T(),
			job.WithEngineSpec(
				model.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
					WithEntrypoint("_start").
					WithParameters(
						"data/hello.txt",
						"does/not/exist.txt",
					).
					Build(),
			),
		),
		ResultsChecker: expectResultsNone,
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}

func (s *DefaultPublisherSuite) TestDefaultPublisher() {
	stack := scenario.StackConfig{}
	stack.DefaultPublisher = "ipfs"

	testcase := scenario.Scenario{
		Spec: testutils.MakeSpecWithOpts(s.T(),
			job.WithEngineSpec(
				model.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
					WithEntrypoint("_start").
					WithParameters(
						"data/hello.txt",
						"does/not/exist.txt",
					).
					Build(),
			),
		),
		Stack:          &stack,
		ResultsChecker: expectResultsSome,
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
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
