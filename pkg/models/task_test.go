//go:build unit || !integration

package models

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TaskTestSuite struct {
	suite.Suite
}

func TestTaskSuite(t *testing.T) {
	suite.Run(t, new(TaskTestSuite))
}

func (suite *TaskTestSuite) TestTaskNormalization() {
	task := &Task{
		Name:      "test-task",
		Engine:    &SpecConfig{Type: "docker"},
		Publisher: &SpecConfig{Type: "s3"},
		InputSources: []*InputSource{
			{Alias: "input1", Target: "/input1", Source: &SpecConfig{Type: "http"}},
		},
		ResultPaths: []*ResultPath{
			{Name: "output1", Path: "/output1"},
		},
		Meta:            nil,
		Env:             nil,
		ResourcesConfig: nil,
		Network:         nil,
		Timeouts:        nil,
	}

	task.Normalize()

	suite.NotNil(task.Meta)
	suite.NotNil(task.Env)
	suite.NotNil(task.ResourcesConfig)
	suite.NotNil(task.Network)
	suite.NotNil(task.Timeouts)
	suite.NotEmpty(task.InputSources)
	suite.NotEmpty(task.ResultPaths)
}

func (suite *TaskTestSuite) TestTaskValidation() {
	type validationMode int

	const (
		noError validationMode = iota
		submissionError
		postSubmissionError
	)

	tests := []struct {
		name           string
		task           *Task
		validationMode validationMode
		errMsg         string
	}{
		{
			name: "Valid task",
			task: &Task{
				Name:   "valid-task",
				Engine: &SpecConfig{Type: "docker"},
				InputSources: []*InputSource{
					{Alias: "input1", Target: "/input1", Source: &SpecConfig{Type: "http"}},
					{Alias: "input2", Target: "/input2", Source: &SpecConfig{Type: "http"}},
				},
				ResultPaths: []*ResultPath{
					{Name: "output1", Path: "/output1"},
					{Name: "output2", Path: "/output2"},
				},
				Publisher: &SpecConfig{Type: "s3"},
			},
			validationMode: noError,
		},
		{
			name: "Empty task name",
			task: &Task{
				Name:   "",
				Engine: &SpecConfig{Type: "docker"},
			},
			validationMode: submissionError,
			errMsg:         "missing task name",
		},
		{
			name: "Duplicate input source alias",
			task: &Task{
				Name:   "duplicate-alias",
				Engine: &SpecConfig{Type: "docker"},
				InputSources: []*InputSource{
					{Alias: "input1", Target: "/input1", Source: &SpecConfig{Type: "http"}},
					{Alias: "input1", Target: "/input2", Source: &SpecConfig{Type: "http"}},
				},
			},
			validationMode: submissionError,
			errMsg:         "input source with alias 'input1' already exists",
		},
		{
			name: "Duplicate input source target",
			task: &Task{
				Name:   "duplicate-target",
				Engine: &SpecConfig{Type: "docker"},
				InputSources: []*InputSource{
					{Alias: "input1", Target: "/input", Source: &SpecConfig{Type: "http"}},
					{Alias: "input2", Target: "/input", Source: &SpecConfig{Type: "http"}},
				},
			},
			validationMode: submissionError,
			errMsg:         "input source with target '/input' already exists",
		},
		{
			name: "Duplicate result path name",
			task: &Task{
				Name:   "duplicate-result-name",
				Engine: &SpecConfig{Type: "docker"},
				ResultPaths: []*ResultPath{
					{Name: "output", Path: "/output1"},
					{Name: "output", Path: "/output2"},
				},
				Publisher: &SpecConfig{Type: "s3"},
			},
			validationMode: submissionError,
			errMsg:         "result path with name 'output' already exists",
		},
		{
			name: "Duplicate result path",
			task: &Task{
				Name:   "duplicate-result-path",
				Engine: &SpecConfig{Type: "docker"},
				ResultPaths: []*ResultPath{
					{Name: "output1", Path: "/output"},
					{Name: "output2", Path: "/output"},
				},
				Publisher: &SpecConfig{Type: "s3"},
			},
			validationMode: submissionError,
			errMsg:         "result path '/output' already exists",
		},
		{
			name: "Result paths without publisher",
			task: &Task{
				Name:   "missing-publisher",
				Engine: &SpecConfig{Type: "docker"},
				ResultPaths: []*ResultPath{
					{Name: "output", Path: "/output"},
				},
			},
			validationMode: postSubmissionError,
			errMsg:         "publisher must be set if result paths are set",
		},
		{
			name: "Misconfigured timeouts",
			task: &Task{
				Name:   "invalid-timeouts",
				Engine: &SpecConfig{Type: "docker"},
				Timeouts: &TimeoutConfig{
					ExecutionTimeout: 100,
					TotalTimeout:     10,
				},
			},
			validationMode: postSubmissionError,
			errMsg:         "should be less than total timeout",
		},
		{
			name: "Invalid timeouts",
			task: &Task{
				Name:   "invalid-timeouts",
				Engine: &SpecConfig{Type: "docker"},
				Timeouts: &TimeoutConfig{
					ExecutionTimeout: -1,
				},
			},
			validationMode: submissionError,
			errMsg:         "task timeouts validation failed",
		},
		{
			name: "Invalid resources",
			task: &Task{
				Name:   "invalid-resources",
				Engine: &SpecConfig{Type: "docker"},
				ResourcesConfig: &ResourcesConfig{
					CPU: "-1",
				},
			},
			validationMode: submissionError,
			errMsg:         "task resources validation failed",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.task.Normalize()

			// Test ValidateSubmission()
			err := tt.task.ValidateSubmission()
			if tt.validationMode == submissionError {
				suite.Error(err)
				suite.Contains(err.Error(), tt.errMsg)
			} else {
				suite.NoError(err)
			}

			// Test Validate()
			// Should always fail if ValidateSubmission() failed
			err = tt.task.Validate()
			if tt.validationMode != noError {
				suite.Error(err)
				suite.Contains(err.Error(), tt.errMsg)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *TaskTestSuite) TestTaskCopy() {
	original := &Task{
		Name:      "original-task",
		Engine:    &SpecConfig{Type: "docker"},
		Publisher: &SpecConfig{Type: "s3"},
		InputSources: []*InputSource{
			{Alias: "input1", Target: "/input1", Source: &SpecConfig{Type: "http"}},
		},
		ResultPaths: []*ResultPath{
			{Name: "output1", Path: "/output1"},
		},
		Meta: map[string]string{"key": "value"},
		Env:  map[string]string{"ENV_VAR": "value"},
	}

	cpy := original.Copy()

	suite.Equal(original, cpy, "The task and its copy should be deeply equal")
	suite.NotSame(original, cpy, "The task and its copy should not be the same instance")

	// Ensure nested objects are deeply copied
	suite.NotSame(original.Engine, cpy.Engine, "The Engine in the task and its copy should not be the same instance")
	suite.NotSame(original.Publisher, cpy.Publisher, "The Publisher in the task and its copy should not be the same instance")
	for i := range original.InputSources {
		suite.NotSame(original.InputSources[i], cpy.InputSources[i], "The InputSources in the task and its copy should not be the same instance")
	}
	for i := range original.ResultPaths {
		suite.NotSame(original.ResultPaths[i], cpy.ResultPaths[i], "The ResultPaths in the task and its copy should not be the same instance")
	}

	// Ensure it's a deep copy by modifying the copy
	cpy.Name = "modified-task"
	cpy.Meta["new_key"] = "new_value"
	cpy.Env["NEW_ENV_VAR"] = "new_value"

	suite.NotEqual(original.Name, cpy.Name)
	suite.NotEqual(original.Meta, cpy.Meta)
	suite.NotEqual(original.Env, cpy.Env)
}

func (suite *TaskTestSuite) TestAllStorageTypes() {
	task := &Task{
		InputSources: []*InputSource{
			{Source: &SpecConfig{Type: "s3"}},
			{Source: &SpecConfig{Type: "url"}},
			{Source: &SpecConfig{Type: "s3"}}, // Duplicate to test uniqueness
		},
	}

	storageTypes := task.AllStorageTypes()
	suite.ElementsMatch([]string{"s3", "url"}, storageTypes)
	suite.Len(storageTypes, 2, "Should return only unique storage types")
}
