//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"

	"github.com/stretchr/testify/suite"
)

type DefaultPublisherSuite struct {
	scenario.ScenarioRunner
}

func TestDefaultPublisherSuite(t *testing.T) {
	suite.Run(t, new(DefaultPublisherSuite))
}

func getTestEngine() *models.SpecConfig {
	return &models.SpecConfig{
		Type: models.EngineDocker,
		Params: dockmodels.EngineSpec{
			Image: "ubuntu:latest",
			Entrypoint: []string{"/bin/bash", "-c", `
                echo "output to stdout" && \
                if [ ! -d /outputs ]; then \
                    mkdir -p /outputs; \
                fi && \
                echo "output to file" > /outputs/test.txt
            `},
		}.ToMap(),
	}
}

// TestNoDefaultPublisherNoResultPath verifies that when no default publisher
// and no result path are defined, the job succeeds but produces no results
func (s *DefaultPublisherSuite) TestNoDefaultPublisherNoResultPath() {
	testcase := scenario.Scenario{
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name:   s.T().Name(),
					Engine: getTestEngine(),
				},
			},
		},
		ResultsChecker: scenario.FileNotExists(downloader.DownloadFilenameStdout),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}

// TestNoDefaultPublisherWithResultPath verifies that when no default publisher
// is defined but result path is specified, the job fails
func (s *DefaultPublisherSuite) TestNoDefaultPublisherWithResultPath() {
	testcase := scenario.Scenario{
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name:   s.T().Name(),
					Engine: getTestEngine(),
					ResultPaths: []*models.ResultPath{
						{
							Name: "outputs",
							Path: "/outputs",
						},
					},
				},
			},
		},
		SubmitChecker: scenario.SubmitJobFail(),
	}

	s.RunScenario(testcase)
}

// TestDefaultPublisherNoResultPath verifies that when default publisher is defined
// but no result path is specified, only stdout is captured
func (s *DefaultPublisherSuite) TestDefaultPublisherNoResultPath() {
	testcase := scenario.Scenario{
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name:   s.T().Name(),
					Engine: getTestEngine(),
				},
			},
		},
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithDefaultPublisher(types.DefaultPublisherConfig{
					Type:   models.PublisherLocal,
					Params: make(map[string]string),
				}),
			},
		},
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(downloader.DownloadFilenameStdout, "output to stdout\n"),
			scenario.FileNotExists("outputs/test.txt"),
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}

// TestDefaultPublisherWithResultPath verifies that when both default publisher
// and result path are defined, both stdout and specified outputs are captured
func (s *DefaultPublisherSuite) TestDefaultPublisherWithResultPath() {
	testcase := scenario.Scenario{
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name:   s.T().Name(),
					Engine: getTestEngine(),
					ResultPaths: []*models.ResultPath{
						{
							Name: "outputs",
							Path: "/outputs",
						},
					},
				},
			},
		},
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithDefaultPublisher(types.DefaultPublisherConfig{
					Type:   models.PublisherLocal,
					Params: make(map[string]string),
				}),
			},
		},
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(downloader.DownloadFilenameStdout, "output to stdout\n"),
			scenario.FileEquals("outputs/test.txt", "output to file\n"),
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}
