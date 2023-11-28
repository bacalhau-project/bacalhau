//go:build unit || !integration

package environment_test

import (
	"context"
	"os"
	"path/filepath"
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

	testcases := []struct {
		name           string
		build_error    bool
		outputs        []*models.ResultPath
		expected_dirs  []string
		expected_files []string
	}{
		{
			name:        "no io",
			build_error: false,
			outputs:     nil,
			expected_dirs: []string{
				"jobid/executionid/logs",
				"jobid/executionid/output",
			},
			expected_files: []string{
				"jobid/executionid/logs/stdout",
				"jobid/executionid/logs/stderr",
			},
		},
		{
			name:        "single resultspath",
			build_error: false,
			outputs: []*models.ResultPath{
				{
					Name: "test",
					Path: "",
				},
			},
			expected_dirs: []string{
				"jobid/executionid/logs",
				"jobid/executionid/output",
				"jobid/executionid/output/test",
			},
			expected_files: []string{
				"jobid/executionid/logs/stdout",
				"jobid/executionid/logs/stderr",
			},
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			data_home, _ := os.MkdirTemp("", "")
			defer os.Remove(data_home)

			execution := &models.Execution{
				ID:    "executionid",
				JobID: "jobid",
				Job:   getJob("test", tc.outputs),
			}
			e := environment.New()

			err := e.Build(context.TODO(), execution, data_home)
			if tc.build_error {
				s.Require().Error(err)
				return
			} else {
				s.Require().NoError(err)
			}

			for _, partPath := range tc.expected_dirs {
				fullpath := filepath.Join(data_home, partPath)
				s.Require().DirExists(fullpath)
			}

			for _, partPath := range tc.expected_files {
				fullpath := filepath.Join(data_home, partPath)
				s.Require().FileExists(fullpath)
			}
		})
	}
}

func getTask(results []*models.ResultPath) *models.Task {
	task := &models.Task{
		ResultPaths: results,
	}

	task.Normalize()
	return task
}

func getJob(jobID string, results []*models.ResultPath) *models.Job {
	job := &models.Job{
		ID:    jobID,
		Tasks: []*models.Task{getTask(results)},
	}

	job.Normalize()
	return job
}
