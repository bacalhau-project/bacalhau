//go:build unit || !integration

package logstream

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

// CompletedStreamerSuite is a suite of tests for the CompletedStreamer
type CompletedStreamerSuite struct {
	suite.Suite
}

// TestSTDOUTOnly tests streaming when only STDOUT is present
func (suite *CompletedStreamerSuite) TestSTDOUTOnly() {
	execution := &models.Execution{
		RunOutput: &models.RunCommandResult{
			STDOUT: "stdout line 1\nstdout line 2",
		},
	}

	streamer := NewCompletedStreamer(CompletedStreamerParams{Execution: execution})
	ch := streamer.Stream(context.Background())

	count := 0
	for log := range ch {
		suite.Nil(log.Err)
		suite.Equal(models.ExecutionLogTypeSTDOUT, log.Value.Type)
		suite.Contains([]string{"stdout line 1", "stdout line 2"}, log.Value.Line)
		count++
	}
	suite.Equal(2, count)
}

// TestSTDERROnly tests streaming when only STDERR is present
func (suite *CompletedStreamerSuite) TestSTDERROnly() {
	execution := &models.Execution{
		RunOutput: &models.RunCommandResult{
			STDERR: "stderr line 1\nstderr line 2",
		},
	}

	streamer := NewCompletedStreamer(CompletedStreamerParams{Execution: execution})
	ch := streamer.Stream(context.Background())

	count := 0
	for log := range ch {
		suite.Nil(log.Err)
		suite.Equal(models.ExecutionLogTypeSTDERR, log.Value.Type)
		suite.Contains([]string{"stderr line 1", "stderr line 2"}, log.Value.Line)
		count++
	}
	suite.Equal(2, count)
}

// TestEmptyOutput tests streaming when there is no output
func (suite *CompletedStreamerSuite) TestEmptyOutput() {
	execution := &models.Execution{
		RunOutput: &models.RunCommandResult{},
	}

	streamer := NewCompletedStreamer(CompletedStreamerParams{Execution: execution})
	ch := streamer.Stream(context.Background())

	count := 0
	for range ch {
		count++
	}
	suite.Equal(0, count)
}

// TestSTDOUTAndSTDERR tests streaming when both STDOUT and STDERR are present
func (suite *CompletedStreamerSuite) TestSTDOUTAndSTDERR() {
	execution := &models.Execution{
		RunOutput: &models.RunCommandResult{
			STDOUT: "stdout line 1\nstdout line 2",
			STDERR: "stderr line 1\nstderr line 2",
		},
	}

	streamer := NewCompletedStreamer(CompletedStreamerParams{Execution: execution})
	ch := streamer.Stream(context.Background())

	stdoutFirst := true
	for log := range ch {
		suite.Nil(log.Err)
		if stdoutFirst {
			suite.Equal(models.ExecutionLogTypeSTDOUT, log.Value.Type)
			stdoutFirst = false
		}
	}

	suite.False(stdoutFirst, "STDOUT should be processed first")
}

// TestCompletedStreamerSuite runs the CompletedStreamer test suite
func TestCompletedStreamerSuite(t *testing.T) {
	suite.Run(t, new(CompletedStreamerSuite))
}
