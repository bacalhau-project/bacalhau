package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type ExecutionTestSuite struct {
	suite.Suite
}

func (suite *ExecutionTestSuite) TestExecutionNormalization() {
	execution := &models.Execution{
		ID:                 "test-execution",
		Namespace:          "",
		Job:                nil,
		AllocatedResources: nil,
		PublishedResult:    nil,
		RunOutput:          nil,
	}

	execution.Normalize()

	// fields that should be set to default values
	suite.NotNil(execution.AllocatedResources)
	suite.NotNil(execution.PublishedResult)
	suite.NotNil(execution.RunOutput)

	// fields that should remain nil/empty
	suite.Nil(execution.Job)
	suite.Empty(execution.Namespace)
}

func (suite *ExecutionTestSuite) TestExecutionValidation() {
	execution := &models.Execution{
		ID:        "invalid execution id",
		Namespace: "",
		JobID:     "",
	}

	err := execution.Validate()
	suite.Error(err)
	suite.Contains(err.Error(), "execution ID contains a space")
	suite.Contains(err.Error(), "execution must be in a namespace")
	suite.Contains(err.Error(), "missing execution job ID")
}

func (suite *ExecutionTestSuite) TestExecutionCopyPopulated() {
	execution := mock.Execution()

	// Ensure fields are populated for this test
	execution.RunOutput = &models.RunCommandResult{
		STDOUT:          "test output",
		StdoutTruncated: false,
		STDERR:          "test error",
		StderrTruncated: false,
		ExitCode:        0,
		ErrorMsg:        "",
	}

	execution.PublishedResult = &models.SpecConfig{
		Type: "test-result",
		Params: map[string]interface{}{
			"key": "value",
		},
	}

	execution.AllocatedResources = &models.AllocatedResources{
		Tasks: map[string]*models.Resources{
			"task1": {
				CPU:    1.0,
				Memory: 1024,
				Disk:   10240,
				GPU:    1,
			},
		},
	}

	cpy := execution.Copy()

	suite.NotNil(cpy)
	suite.Equal(execution, cpy, "The execution and its copy should be deeply equal")
	suite.NotSame(execution, cpy, "The execution and its copy should not be the same instance")

	// Check RunOutput
	suite.NotNil(cpy.RunOutput)
	suite.Equal(execution.RunOutput, cpy.RunOutput, "RunOutput should have equal values")
	suite.NotSame(execution.RunOutput, cpy.RunOutput, "RunOutput should not be the same instance")

	// Check PublishedResult
	suite.NotNil(cpy.PublishedResult)
	suite.Equal(execution.PublishedResult, cpy.PublishedResult, "PublishedResult should have equal values")
	suite.NotSame(execution.PublishedResult, cpy.PublishedResult, "PublishedResult should not be the same instance")

	// Check AllocatedResources
	suite.NotNil(cpy.AllocatedResources)
	suite.Equal(execution.AllocatedResources, cpy.AllocatedResources, "AllocatedResources should have equal values")
	suite.NotSame(execution.AllocatedResources, cpy.AllocatedResources, "AllocatedResources should not be the same instance")
	suite.NotSame(execution.AllocatedResources.Tasks["task1"], cpy.AllocatedResources.Tasks["task1"], "AllocatedResources tasks should not be the same instance")

	// Modify the copy to ensure it doesn't affect the original
	cpy.RunOutput.STDOUT = "modified output"
	cpy.PublishedResult.Params["key"] = "modified value"
	cpy.AllocatedResources.Tasks["task1"].CPU = 2.0

	suite.NotEqual(execution.RunOutput.STDOUT, cpy.RunOutput.STDOUT, "Modifying the copy's RunOutput should not affect the original")
	suite.NotEqual(execution.PublishedResult.Params["key"], cpy.PublishedResult.Params["key"], "Modifying the copy's PublishedResult should not affect the original")
	suite.NotEqual(execution.AllocatedResources.Tasks["task1"].CPU, cpy.AllocatedResources.Tasks["task1"].CPU, "Modifying the copy's AllocatedResources should not affect the original")
}

func (suite *ExecutionTestSuite) TestExecutionCopyNilFields() {
	execution := mock.Execution()

	// Ensure fields are nil for this test
	execution.RunOutput = nil
	execution.PublishedResult = nil
	execution.AllocatedResources = nil

	cpy := execution.Copy()

	suite.NotNil(cpy)
	suite.Equal(execution, cpy, "The execution and its copy should be deeply equal")
	suite.NotSame(execution, cpy, "The execution and its copy should not be the same instance")

	// Check that nil fields are preserved in the copy
	suite.Nil(cpy.RunOutput, "RunOutput should be nil in the copy")
	suite.Nil(cpy.PublishedResult, "PublishedResult should be nil in the copy")
	suite.Nil(cpy.AllocatedResources, "AllocatedResources should be nil in the copy")

	// Modify the copy by setting non-nil values
	cpy.RunOutput = &models.RunCommandResult{STDOUT: "new output"}
	cpy.PublishedResult = &models.SpecConfig{Type: "new-result"}
	cpy.AllocatedResources = &models.AllocatedResources{
		Tasks: map[string]*models.Resources{
			"new-task": {CPU: 1.0},
		},
	}

	// Ensure the original execution is not affected
	suite.Nil(execution.RunOutput, "Modifying the copy's RunOutput should not affect the original")
	suite.Nil(execution.PublishedResult, "Modifying the copy's PublishedResult should not affect the original")
	suite.Nil(execution.AllocatedResources, "Modifying the copy's AllocatedResources should not affect the original")
}

func (suite *ExecutionTestSuite) TestExecutionJobNamespacedID() {
	execution := mock.Execution()
	nsID := execution.JobNamespacedID()
	suite.Equal(execution.JobID, nsID.ID)
	suite.Equal(execution.Namespace, nsID.Namespace)
}

func (suite *ExecutionTestSuite) TestExecutionIsExpired() {
	now := time.Now()
	execution := mock.Execution()
	execution.ComputeState.StateType = models.ExecutionStateBidAccepted
	execution.ModifyTime = now.Add(-1 * time.Hour).UnixNano()

	suite.True(execution.IsExpired(now))

	execution.ComputeState.StateType = models.ExecutionStateCompleted
	suite.False(execution.IsExpired(now))

	execution.ComputeState.StateType = models.ExecutionStateBidAccepted
	execution.ModifyTime = now.Add(1 * time.Hour).UnixNano()
	suite.False(execution.IsExpired(now))
}

func (suite *ExecutionTestSuite) TestExecutionIsTerminalState() {
	execution := mock.Execution()

	terminalStates := []models.ExecutionStateType{
		models.ExecutionStateCompleted,
		models.ExecutionStateFailed,
		models.ExecutionStateCancelled,
		models.ExecutionStateAskForBidRejected,
		models.ExecutionStateBidRejected,
	}

	for _, state := range terminalStates {
		execution.ComputeState.StateType = state
		suite.True(execution.IsTerminalState())
	}

	nonTerminalStates := []models.ExecutionStateType{
		models.ExecutionStateNew,
		models.ExecutionStateAskForBid,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionStateBidAccepted,
	}

	for _, state := range nonTerminalStates {
		execution.ComputeState.StateType = state
		suite.False(execution.IsTerminalState())
	}
}

func (suite *ExecutionTestSuite) TestExecutionIsDiscarded() {
	execution := mock.Execution()

	discardedStates := []models.ExecutionStateType{
		models.ExecutionStateAskForBidRejected,
		models.ExecutionStateBidRejected,
		models.ExecutionStateCancelled,
		models.ExecutionStateFailed,
	}

	for _, state := range discardedStates {
		execution.ComputeState.StateType = state
		suite.True(execution.IsDiscarded())
	}

	nonDiscardedStates := []models.ExecutionStateType{
		models.ExecutionStateNew,
		models.ExecutionStateAskForBid,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionStateBidAccepted,
		models.ExecutionStateCompleted,
	}

	for _, state := range nonDiscardedStates {
		execution.ComputeState.StateType = state
		suite.False(execution.IsDiscarded())
	}
}

// Run the test suite
func TestExecutionTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutionTestSuite))
}
