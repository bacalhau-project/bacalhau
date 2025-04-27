package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ExecutionsTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestExecutionsSuite(t *testing.T) {
	suite.Run(t, new(ExecutionsTestSuite))
}

func (s *ExecutionsTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *ExecutionsTestSuite) createTestJob() models.Job {
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

func (s *ExecutionsTestSuite) createTestExecution() models.Execution {
	return models.Execution{
		ID:        "test-execution-id",
		JobID:     "test-job-id",
		EvalID:    "test-eval-id",
		NodeID:    "test-node-id",
		Namespace: "test-namespace",
		AllocatedResources: &models.AllocatedResources{
			Tasks: map[string]*models.Resources{
				"test-task": {
					CPU:    1.0,
					Memory: 1024,
					Disk:   2048,
					GPU:    1,
					GPUs: []models.GPU{
						{Name: "test-gpu", Vendor: "test-vendor"},
					},
				},
			},
		},
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateRunning,
			Details: map[string]string{
				models.DetailsKeyErrorCode: "test-error",
			},
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateRunning,
			Message:   "test-message",
			Details: map[string]string{
				models.DetailsKeyErrorCode: "test-error",
			},
		},
		PublishedResult: &models.SpecConfig{
			Type: "test-publisher",
		},
		RunOutput: &models.RunCommandResult{
			StdoutTruncated: true,
			StderrTruncated: false,
			ExitCode:        0,
		},
		PreviousExecution: "prev-execution",
		NextExecution:     "next-execution",
		FollowupEvalID:    "followup-eval",
		Revision:          1,
		CreateTime:        time.Now().UnixNano(),
		ModifyTime:        time.Now().UnixNano(),
	}
}

func (s *ExecutionsTestSuite) TestCreatedExecutionEvent() {
	execution := s.createTestExecution()
	event := NewCreatedExecutionEvent(execution)
	s.Equal(CreatedExecutionEventType, event.Type())

	props := event.Properties()

	// Test basic properties
	s.Equal("test-job-id", props["job_id"])
	s.Equal("test-execution-id", props["execution_id"])
	s.Equal("test-eval-id", props["evaluation_id"])
	s.Equal(hashString("test-node-id"), props["node_name_hash"])
	s.Equal(hashString("test-namespace"), props["namespace_hash"])
	s.Equal(models.ExecutionStateRunning.String(), props["compute_state"])
	s.Equal(uint64(1), props["revision"])
	s.Equal(models.ExecutionDesiredStateRunning.String(), props["desired_state"])

	// Test resource properties
	resources := props["resources"].(map[string]resource)
	hashedTaskName := hashString("test-task")
	taskResources := resources[hashedTaskName]
	s.Equal(1.0, taskResources.CPUUnits)
	s.Equal(uint64(1024), taskResources.MemoryBytes)
	s.Equal(uint64(2048), taskResources.DiskBytes)
	s.Equal(uint64(1), taskResources.GPUCount)
	s.Len(taskResources.GPUTypes, 1)
	s.Equal("test-gpu", taskResources.GPUTypes[0].Name)
	s.Equal("test-vendor", taskResources.GPUTypes[0].Vendor)

	// Test state error codes
	s.Equal("test-error", props["desired_state_error_code"])
	s.Equal("test-error", props["compute_state_error_code"])

	// Test run output properties
	s.Equal(true, props["run_result_stdout_truncated"])
	s.Equal(false, props["run_result_stderr_truncated"])
	s.Equal(0, props["run_result_exit_code"])

	// Test publisher
	s.Equal("test-publisher", props["publisher_type"])

	// Test related IDs
	s.Equal("prev-execution", props["previous_execution"])
	s.Equal("next-execution", props["next_execution"])
	s.Equal("followup-eval", props["followup_eval_id"])

	// Test time fields
	createTime := props["create_time"].(time.Time)
	modifyTime := props["modify_time"].(time.Time)
	s.True(createTime.Before(time.Now()))
	s.True(modifyTime.Before(time.Now()))
}

func (s *ExecutionsTestSuite) TestTerminalExecutionEvent() {
	execution := s.createTestExecution()
	event := NewTerminalExecutionEvent(execution)
	s.Equal(TerminalExecutionEventType, event.Type())

	props := event.Properties()

	// Test basic properties
	s.Equal("test-job-id", props["job_id"])
	s.Equal("test-execution-id", props["execution_id"])
	s.Equal("test-eval-id", props["evaluation_id"])
	s.Equal(hashString("test-node-id"), props["node_name_hash"])
	s.Equal(hashString("test-namespace"), props["namespace_hash"])
	s.Equal(models.ExecutionStateRunning.String(), props["compute_state"])
	s.Equal(uint64(1), props["revision"])
	s.Equal(models.ExecutionDesiredStateRunning.String(), props["desired_state"])

	// Test resource properties
	resources := props["resources"].(map[string]resource)
	hashedTaskName := hashString("test-task")
	taskResources := resources[hashedTaskName]
	s.Equal(1.0, taskResources.CPUUnits)
	s.Equal(uint64(1024), taskResources.MemoryBytes)
	s.Equal(uint64(2048), taskResources.DiskBytes)
	s.Equal(uint64(1), taskResources.GPUCount)
	s.Len(taskResources.GPUTypes, 1)
	s.Equal("test-gpu", taskResources.GPUTypes[0].Name)
	s.Equal("test-vendor", taskResources.GPUTypes[0].Vendor)

	// Test state error codes
	s.Equal("test-error", props["desired_state_error_code"])
	s.Equal("test-error", props["compute_state_error_code"])

	// Test run output properties
	s.Equal(true, props["run_result_stdout_truncated"])
	s.Equal(false, props["run_result_stderr_truncated"])
	s.Equal(0, props["run_result_exit_code"])

	// Test publisher
	s.Equal("test-publisher", props["publisher_type"])

	// Test related IDs
	s.Equal("prev-execution", props["previous_execution"])
	s.Equal("next-execution", props["next_execution"])
	s.Equal("followup-eval", props["followup_eval_id"])

	// Test time fields
	createTime := props["create_time"].(time.Time)
	modifyTime := props["modify_time"].(time.Time)
	s.True(createTime.Before(time.Now()))
	s.True(modifyTime.Before(time.Now()))
}

func (s *ExecutionsTestSuite) TestComputeMessageExecutionEvent() {
	execution := s.createTestExecution()
	event := NewComputeMessageExecutionEvent(execution)
	s.Equal(ComputeMessageExecutionEventType, event.Type())

	props := event.Properties()
	s.Equal("test-job-id", props["job_id"])
	s.Equal("test-execution-id", props["execution_id"])
	s.Equal("test-message", props["compute_message"])
	s.Equal("test-error", props["compute_state_error_code"])
}

func (s *ExecutionsTestSuite) TestEmptyExecutionEvent() {
	emptyExecution := models.Execution{}
	emptyExecution.Normalize()
	event := NewCreatedExecutionEvent(emptyExecution)
	props := event.Properties()

	// Test that empty fields are handled properly
	s.Equal("", props["job_id"])
	s.Equal("", props["execution_id"])
	s.Equal("", props["node_name_hash"])
	s.Equal(models.ExecutionStateUndefined.String(), props["compute_state"])
	s.Equal(uint64(0), props["revision"])
	s.Equal(models.ExecutionDesiredStatePending.String(), props["desired_state"])

	// Test that resources map is empty
	resources := props["resources"].(map[string]resource)
	s.Empty(resources)

	// Test that error codes are empty
	s.Equal("", props["desired_state_error_code"])
	s.Equal("", props["compute_state_error_code"])

	// Test that run output properties are false/0
	s.Equal(false, props["run_result_stdout_truncated"])
	s.Equal(false, props["run_result_stderr_truncated"])
	s.Equal(0, props["run_result_exit_code"])

	// Test that publisher is empty
	s.Equal("", props["publisher_type"])

	// Test that related IDs are empty
	s.Equal("", props["previous_execution"])
	s.Equal("", props["next_execution"])
	s.Equal("", props["followup_eval_id"])
}

func (s *ExecutionsTestSuite) TestNilResourcesExecutionEvent() {
	execution := s.createTestExecution()
	execution.AllocatedResources = nil
	event := NewCreatedExecutionEvent(execution)
	props := event.Properties()

	// Test that resources map is empty when AllocatedResources is nil
	resources := props["resources"].(map[string]resource)
	s.Empty(resources)
}

func (s *ExecutionsTestSuite) TestNilRunOutputExecutionEvent() {
	execution := s.createTestExecution()
	execution.RunOutput = nil
	event := NewCreatedExecutionEvent(execution)
	props := event.Properties()

	// Test that run output properties are false/0 when RunOutput is nil
	s.Equal(false, props["run_result_stdout_truncated"])
	s.Equal(false, props["run_result_stderr_truncated"])
	s.Equal(0, props["run_result_exit_code"])
}

func (s *ExecutionsTestSuite) TestNilStateDetailsExecutionEvent() {
	execution := s.createTestExecution()
	execution.DesiredState.Details = nil
	execution.ComputeState.Details = nil
	event := NewCreatedExecutionEvent(execution)
	props := event.Properties()

	// Test that error codes are empty when Details is nil
	s.Equal("", props["desired_state_error_code"])
	s.Equal("", props["compute_state_error_code"])
}
