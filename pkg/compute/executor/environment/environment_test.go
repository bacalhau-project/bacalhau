package environment_test

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/executor/environment"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

type EnvironmentTestSuite struct {
	suite.Suite
}

func TestEnvironmentTestSuite(t *testing.T) {
	suite.Run(t, new(EnvironmentTestSuite))
}

func (s *EnvironmentTestSuite) TestPathStructures() {
	execution := &models.Execution{
		ID:    "executionid",
		JobID: "jobid",
		Job: &models.Job{
			ID: "jobid",
			Tasks: []*models.Task{
				{
					ResultPaths: []*models.ResultPath{
						{
							Name: "Test",
							Path: "/outputs/something",
						},
					},
				},
			},
		},
	}
	e := environment.New()

	if execution.Job.Tasks[0].InputSources == nil {
		execution.Job.Tasks[0].InputSources = []*models.InputSource{}
	}

	err := e.Build(context.TODO(), execution)
	s.Require().NoError(err)
}
