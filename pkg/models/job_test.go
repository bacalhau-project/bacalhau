//go:build unit || !integration

package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type JobTestSuite struct {
	suite.Suite
}

func (suite *JobTestSuite) TestJobNormalization() {
	job := &models.Job{
		ID:          "test-job",
		Name:        "",
		Namespace:   "",
		Meta:        nil,
		Labels:      nil,
		Constraints: nil,
		Tasks:       nil,
	}

	job.Normalize()

	suite.Equal(models.DefaultNamespace, job.Namespace)
	suite.Equal("test-job", job.Name)
	suite.NotNil(job.Meta)
	suite.NotNil(job.Labels)
	suite.NotNil(job.Constraints)
	suite.NotNil(job.Tasks)
}

func (suite *JobTestSuite) TestJobValidation() {
	job := &models.Job{
		ID:        "invalid job id",
		Name:      "",
		Namespace: "",
	}

	err := job.Validate()
	suite.Error(err)
	suite.Contains(err.Error(), "missing job name")
	suite.Contains(err.Error(), "job ID contains a space")
	suite.Contains(err.Error(), "job must be in a namespace")
}

func (suite *JobTestSuite) TestJobSanitization() {
	job := &models.Job{
		ID:         "test-job",
		Name:       "test-job",
		Namespace:  "default",
		State:      models.State[models.JobStateType]{StateType: models.JobStateTypeRunning},
		Revision:   1,
		Version:    1,
		CreateTime: time.Now().UnixNano(),
		ModifyTime: time.Now().UnixNano(),
		Tasks: []*models.Task{
			{
				Name: "test-task",
			},
		},
	}

	warnings := job.SanitizeSubmission()

	suite.NotEmpty(warnings)
	suite.Equal(models.JobStateTypeUndefined, job.State.StateType)
	suite.Equal(uint64(0), job.Revision)
	suite.Equal(uint64(0), job.Version)
	suite.Equal(int64(0), job.CreateTime)
	suite.Equal(int64(0), job.ModifyTime)
}

func (suite *JobTestSuite) TestJobCopy() {
	job := mock.Job()
	cpy := job.Copy()

	suite.NotNil(cpy)
	suite.Equal(job, cpy, "The job and its copy should be deeply equal")
	suite.NotSame(job, cpy, "The job and its copy should not be the same instance")

	// Ensure nested objects are deeply copied
	for i, task := range job.Tasks {
		suite.NotSame(task, cpy.Tasks[i], "The tasks in the job and its copy should not be the same instance")
	}
}

func (suite *JobTestSuite) TestIsTerminal() {
	job := &models.Job{
		State: models.State[models.JobStateType]{StateType: models.JobStateTypeCompleted},
	}
	suite.True(job.IsTerminal())

	job.State.StateType = models.JobStateTypeFailed
	suite.True(job.IsTerminal())

	job.State.StateType = models.JobStateTypeStopped
	suite.True(job.IsTerminal())

	job.State.StateType = models.JobStateTypeRunning
	suite.False(job.IsTerminal())
}

func (suite *JobTestSuite) TestNamespacedID() {
	job := mock.Job()
	nsID := job.NamespacedID()
	suite.Equal(job.ID, nsID.ID)
	suite.Equal(job.Namespace, nsID.Namespace)
}

func (suite *JobTestSuite) TestAllStorageTypes() {
	job := mock.Job()
	job.Tasks = []*models.Task{
		{
			InputSources: []*models.InputSource{
				{
					Source: &models.SpecConfig{
						Type: "s3",
					},
				},
				{
					Source: &models.SpecConfig{
						Type: "url",
					},
				},
			},
		},
	}

	storageTypes := job.AllStorageTypes()
	suite.ElementsMatch([]string{"s3", "url"}, storageTypes)
}

// Run the test suite
func TestJobTestSuite(t *testing.T) {
	suite.Run(t, new(JobTestSuite))
}
