//go:build unit || !integration

package models_test

import (
	"encoding/json"
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
	testCases := []struct {
		name          string
		jobType       string
		count         int
		expectedCount int
	}{
		// Test that Normalize doesn't change count for batch/service jobs
		{
			name:          "batch job with count=0",
			jobType:       models.JobTypeBatch,
			count:         0,
			expectedCount: 0,
		},
		{
			name:          "service job with count=0",
			jobType:       models.JobTypeService,
			count:         0,
			expectedCount: 0,
		},
		{
			name:          "batch job with count=2",
			jobType:       models.JobTypeBatch,
			count:         2,
			expectedCount: 2,
		},
		{
			name:          "service job with count=3",
			jobType:       models.JobTypeService,
			count:         3,
			expectedCount: 3,
		},

		// Test that Normalize forces count=0 for daemon and ops jobs
		{
			name:          "ops job with count=0",
			jobType:       models.JobTypeOps,
			count:         0,
			expectedCount: 0,
		},
		{
			name:          "daemon job with count=0",
			jobType:       models.JobTypeDaemon,
			count:         0,
			expectedCount: 0,
		},
		{
			name:          "ops job with count=4",
			jobType:       models.JobTypeOps,
			count:         4,
			expectedCount: 0,
		},
		{
			name:          "daemon job with count=5",
			jobType:       models.JobTypeDaemon,
			count:         5,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			job := &models.Job{
				ID:          "test-job",
				Type:        tc.jobType,
				Name:        "",
				Namespace:   "",
				Meta:        nil,
				Labels:      nil,
				Constraints: nil,
				Tasks:       nil,
			}
			job.Count = tc.count

			job.Normalize()

			suite.Equal(models.DefaultNamespace, job.Namespace)
			suite.Equal("test-job", job.Name)
			suite.Equal(tc.jobType, job.Type)
			suite.Equal(tc.expectedCount, job.Count)
			suite.NotNil(job.Meta)
			suite.NotNil(job.Labels)
			suite.NotNil(job.Constraints)
			suite.NotNil(job.Tasks)
		})
	}
}

func (suite *JobTestSuite) TestJobValidation() {
	testCases := []struct {
		name        string
		job         *models.Job
		expectError bool
		errorMsgs   []string
	}{
		{
			name: "invalid job id and missing fields",
			job: &models.Job{
				ID:        "invalid job id",
				Name:      "",
				Namespace: "",
			},
			expectError: true,
			errorMsgs: []string{
				"missing job name",
				"job ID contains a space",
				"job must be in a namespace",
			},
		},
		{
			name: "negative count",
			job: &models.Job{
				ID:        "test-job",
				Name:      "test-job",
				Namespace: "default",
				Type:      models.JobTypeBatch,
				Count:     -1,
			},
			expectError: true,
			errorMsgs: []string{
				"job count must be >= 0",
			},
		},
		{
			name: "daemon job with count > 1",
			job: &models.Job{
				ID:        "test-job",
				Name:      "test-job",
				Namespace: "default",
				Type:      models.JobTypeDaemon,
				Count:     2,
			},
			expectError: true,
			errorMsgs: []string{
				"daemon jobs cannot specify count > 1",
			},
		},
		{
			name: "ops job with count > 1",
			job: &models.Job{
				ID:        "test-job",
				Name:      "test-job",
				Namespace: "default",
				Type:      models.JobTypeOps,
				Count:     3,
			},
			expectError: true,
			errorMsgs: []string{
				"ops jobs cannot specify count > 1",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.job.Validate()
			if tc.expectError {
				suite.Error(err)
				for _, msg := range tc.errorMsgs {
					suite.Contains(err.Error(), msg)
				}
			} else {
				suite.NoError(err)
			}
		})
	}
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
	for i := range job.Tasks {
		suite.NotSame(job.Tasks[i], cpy.Tasks[i], "The tasks in the job and its copy should not be the same instance")
	}
}

func (suite *JobTestSuite) TestJobTask() {
	job := mock.Job()
	suite.Equal(job.Tasks[0], job.Task())
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

// Add new test for JSON marshaling and unmarshaling behavior
func (suite *JobTestSuite) TestJobJSONHandling() {
	testCases := []struct {
		name          string
		jobType       string
		count         *int // nil means omit from JSON
		expectedCount int
	}{
		// Test batch jobs
		{
			name:          "batch job with count=2",
			jobType:       models.JobTypeBatch,
			count:         ptr(2),
			expectedCount: 2,
		},
		{
			name:          "batch job with count=0",
			jobType:       models.JobTypeBatch,
			count:         ptr(0),
			expectedCount: 0,
		},
		{
			name:          "batch job with count omitted",
			jobType:       models.JobTypeBatch,
			count:         nil,
			expectedCount: 1, // Default value for batch jobs
		},
		// Test service jobs
		{
			name:          "service job with count=3",
			jobType:       models.JobTypeService,
			count:         ptr(3),
			expectedCount: 3,
		},
		{
			name:          "service job with count=0",
			jobType:       models.JobTypeService,
			count:         ptr(0),
			expectedCount: 0,
		},
		{
			name:          "service job with count omitted",
			jobType:       models.JobTypeService,
			count:         nil,
			expectedCount: 1, // Default value for service jobs
		},
		// Test daemon jobs
		{
			name:          "daemon job with count=1",
			jobType:       models.JobTypeDaemon,
			count:         ptr(1),
			expectedCount: 1,
		},
		{
			name:          "daemon job with count omitted",
			jobType:       models.JobTypeDaemon,
			count:         nil,
			expectedCount: 0, // Will be set to 0 during normalization
		},
		// Test ops jobs
		{
			name:          "ops job with count=1",
			jobType:       models.JobTypeOps,
			count:         ptr(1),
			expectedCount: 1,
		},
		{
			name:          "ops job with count omitted",
			jobType:       models.JobTypeOps,
			count:         nil,
			expectedCount: 0, // Will be set to 0 during normalization
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create a job with minimal fields
			originalJob := map[string]interface{}{
				"id":        "test-job",
				"name":      "test-job",
				"namespace": "default",
				"type":      tc.jobType,
			}
			if tc.count != nil {
				originalJob["count"] = *tc.count
			}

			// Marshal to JSON
			jsonData, err := json.Marshal(originalJob)
			suite.NoError(err)

			// Unmarshal back to Job
			var job models.Job
			err = json.Unmarshal(jsonData, &job)
			suite.NoError(err)

			// Verify the count
			suite.Equal(tc.expectedCount, job.Count, "Count mismatch for %s", tc.name)
		})
	}
}

func (suite *JobTestSuite) TestJobStateTypeIsRerunnable() {
	testCases := []struct {
		name     string
		state    models.JobStateType
		expected bool
	}{
		{
			name:     "pending state should be rerunnable",
			state:    models.JobStateTypePending,
			expected: true,
		},
		{
			name:     "queued state should be rerunnable",
			state:    models.JobStateTypeQueued,
			expected: true,
		},
		{
			name:     "undefined state should be rerunnable",
			state:    models.JobStateTypeUndefined,
			expected: true,
		},
		{
			name:     "running state should not be rerunnable",
			state:    models.JobStateTypeRunning,
			expected: false,
		},
		{
			name:     "completed state should not be rerunnable",
			state:    models.JobStateTypeCompleted,
			expected: false,
		},
		{
			name:     "failed state should not be rerunnable",
			state:    models.JobStateTypeFailed,
			expected: false,
		},
		{
			name:     "stopped state should not be rerunnable",
			state:    models.JobStateTypeStopped,
			expected: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := tc.state.IsRerunnable()
			suite.Equal(tc.expected, result, "IsRerunnable() result mismatch for state %s", tc.state.String())
		})
	}
}

func (suite *JobTestSuite) TestJobIsRerunnable() {
	testCases := []struct {
		name     string
		state    models.JobStateType
		expected bool
	}{
		{
			name:     "job with pending state should be rerunnable",
			state:    models.JobStateTypePending,
			expected: true,
		},
		{
			name:     "job with queued state should be rerunnable",
			state:    models.JobStateTypeQueued,
			expected: true,
		},
		{
			name:     "job with undefined state should be rerunnable",
			state:    models.JobStateTypeUndefined,
			expected: true,
		},
		{
			name:     "job with running state should not be rerunnable",
			state:    models.JobStateTypeRunning,
			expected: false,
		},
		{
			name:     "job with completed state should not be rerunnable",
			state:    models.JobStateTypeCompleted,
			expected: false,
		},
		{
			name:     "job with failed state should not be rerunnable",
			state:    models.JobStateTypeFailed,
			expected: false,
		},
		{
			name:     "job with stopped state should not be rerunnable",
			state:    models.JobStateTypeStopped,
			expected: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create a standalone job for this test
			job := &models.Job{
				ID:        "rerunnable-test-job-" + tc.state.String(),
				Name:      "rerunnable-test-job",
				Namespace: "test-namespace",
				Type:      models.JobTypeBatch,
				Count:     1,
				State:     models.State[models.JobStateType]{StateType: tc.state},
				Tasks: []*models.Task{
					{
						Name: "test-task",
					},
				},
			}

			result := job.IsRerunnable()
			suite.Equal(tc.expected, result, "Job.IsRerunnable() result mismatch for job with state %s", tc.state.String())
		})
	}
}

// Helper function to create pointer to int
func ptr(i int) *int {
	return &i
}

// Run the test suite
func TestJobTestSuite(t *testing.T) {
	suite.Run(t, new(JobTestSuite))
}
