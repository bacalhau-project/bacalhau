package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestJobSubmitSuite(t *testing.T) {
	suite.Run(t, new(JobTestSuite))
}

func (s *JobTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *JobTestSuite) createTestJob() models.Job {
	return models.Job{
		ID:         "test-job-id",
		Name:       "test-job",
		Namespace:  "test-namespace",
		Type:       "test-type",
		Count:      1,
		Labels:     map[string]string{"key": "value"},
		Meta:       map[string]string{"meta": "value"},
		Version:    1,
		Revision:   1,
		CreateTime: time.Now().UnixNano(),
		ModifyTime: time.Now().UnixNano(),
		Tasks: []*models.Task{
			{
				Name: "test-task",
				Engine: &models.SpecConfig{
					Type: models.EngineDocker,
					Params: map[string]interface{}{
						"Image": "ghcr.io/bacalhau-project/test:latest",
					},
				},
				Publisher: &models.SpecConfig{
					Type: "test-publisher",
				},
				Env: map[string]models.EnvVarValue{
					"ENV": "test",
				},
				Meta: map[string]string{"task-meta": "value"},
				InputSources: []*models.InputSource{
					{Source: &models.SpecConfig{Type: "test-source"}},
				},
				ResultPaths: []*models.ResultPath{
					{Name: "result1"},
					{Name: "result2"},
				},
				Network: &models.NetworkConfig{
					Type:    models.NetworkHTTP,
					Domains: []string{"test.com"},
				},
				Timeouts: &models.TimeoutConfig{
					ExecutionTimeout: 3600,
					QueueTimeout:     1800,
					TotalTimeout:     7200,
				},
			},
		},
	}
}

func (s *JobTestSuite) TestSubmitJobEvent() {
	job := s.createTestJob()

	jobID := "actual-job-id"
	submissionErr := "submission error"
	event := NewSubmitJobEvent(job, "actual-job-id", errors.New(submissionErr))
	s.Equal(SubmitJobEventType, event.Type())

	props := event.Properties()

	// Test basic job properties
	s.Equal(jobID, props["job_id"])
	s.True(props["name_set"].(bool))
	s.Equal(hashString("test-namespace"), props["namespace_hash"])
	s.Equal("test-type", props["type"])
	s.Equal(1, props["count"])
	s.Equal(1, props["labels_count"])
	s.Equal(1, props["meta_count"])
	s.Equal(uint64(1), props["version"])
	s.Equal(uint64(1), props["revision"])

	// Test task properties
	s.Equal(hashString("test-task"), props["task_name_hash"])
	s.Equal(models.EngineDocker, props["task_engine_type"])
	s.Equal("test-publisher", props["task_publisher_type"])
	s.Equal(1, props["task_env_var_count"])
	s.Equal(1, props["task_meta_count"])
	s.Equal([]string{"test-source"}, props["task_input_source_types"])
	s.Equal(2, props["task_result_path_count"])
	s.Equal("ghcr.io/bacalhau-project/test:latest", props["task_docker_image"])

	// Test network properties
	s.Equal(models.NetworkHTTP.String(), props["task_network_type"])
	s.Equal(1, props["task_domains_count"])

	// Test error properties
	s.Equal(submissionErr, props["error"])

	// Test timeout properties
	s.Equal(int64(3600), props["task_execution_timeout"])
	s.Equal(int64(1800), props["task_queue_timeout"])
	s.Equal(int64(7200), props["task_total_timeout"])

	// Test time fields
	createTime := props["create_time"].(time.Time)
	modifyTime := props["modify_time"].(time.Time)
	s.True(createTime.Before(time.Now()))
	s.True(modifyTime.Before(time.Now()))
}

func (s *JobTestSuite) TestSubmitJobEventWithWarnings() {
	job := s.createTestJob()
	warnings := []string{"warning1", "warning2"}
	event := NewSubmitJobEvent(job, job.ID, nil, warnings...)
	s.Equal(SubmitJobEventType, event.Type())

	props := event.Properties()
	s.Equal(warnings, props["warnings"])
}

func (s *JobTestSuite) TestJobTerminalEvent() {
	job := s.createTestJob()
	job.State = models.NewJobState(models.JobStateTypeCompleted)
	event := NewJobTerminalEvent(job)
	s.Equal(TerminalJobEventType, event.Type())

	props := event.Properties()

	// Test basic job properties
	s.Equal("test-job-id", props["job_id"])
	s.True(props["name_set"].(bool))
	s.Equal(hashString("test-namespace"), props["namespace_hash"])
	s.Equal("test-type", props["type"])
	s.Equal(1, props["count"])
	s.Equal(1, props["labels_count"])
	s.Equal(1, props["meta_count"])
	s.Equal(uint64(1), props["version"])
	s.Equal(uint64(1), props["revision"])
	s.Equal(models.JobStateTypeCompleted.String(), props["state"])

	// Test task properties
	s.Equal(hashString("test-task"), props["task_name_hash"])
	s.Equal(models.EngineDocker, props["task_engine_type"])
	s.Equal("test-publisher", props["task_publisher_type"])
	s.Equal(1, props["task_env_var_count"])
	s.Equal(1, props["task_meta_count"])
	s.Equal([]string{"test-source"}, props["task_input_source_types"])
	s.Equal(2, props["task_result_path_count"])
	s.Equal("ghcr.io/bacalhau-project/test:latest", props["task_docker_image"])

	// Test network properties
	s.Equal(models.NetworkHTTP.String(), props["task_network_type"])
	s.Equal(1, props["task_domains_count"])

	// Test timeout properties
	s.Equal(int64(3600), props["task_execution_timeout"])
	s.Equal(int64(1800), props["task_queue_timeout"])
	s.Equal(int64(7200), props["task_total_timeout"])

	// Test time fields
	createTime := props["create_time"].(time.Time)
	modifyTime := props["modify_time"].(time.Time)
	s.True(createTime.Before(time.Now()))
	s.True(modifyTime.Before(time.Now()))
}

func (s *JobTestSuite) TestEmptyJobEvent() {
	emptyJob := models.Job{}
	emptyJob.Normalize()
	event := NewSubmitJobEvent(emptyJob, "", nil)
	props := event.Properties()
	s.Empty(props)
}

func (s *JobTestSuite) TestNilTaskJobEvent() {
	job := s.createTestJob()
	job.Tasks = nil
	event := NewSubmitJobEvent(job, "", nil)
	props := event.Properties()
	s.Empty(props)
}
