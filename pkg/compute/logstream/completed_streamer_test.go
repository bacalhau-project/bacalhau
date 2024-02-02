//go:build unit || !integration

package logstream

import (
	"context"
	"testing"
	"time"

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
		suite.Contains([]string{"stdout line 1\n", "stdout line 2\n"}, log.Value.Line)
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
		suite.Contains([]string{"stderr line 1\n", "stderr line 2\n"}, log.Value.Line)
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

// TestContextCancellation tests the streamer's response to a cancelled context
func (suite *CompletedStreamerSuite) TestContextCancellation() {
	execution := &models.Execution{
		RunOutput: &models.RunCommandResult{
			STDOUT: "stdout line 1\nstdout line 2\nstdout line 3",
			STDERR: "stderr line 1\nstderr line 2\nstderr line 3",
		},
	}

	streamer := NewCompletedStreamer(CompletedStreamerParams{Execution: execution})
	ctx, cancel := context.WithCancel(context.Background())

	ch := streamer.Stream(ctx)

	// Wait for the first line to be enqueued in the channel
	time.Sleep(100 * time.Millisecond)

	// cancel the context after a single line has been enqueued
	cancel()

	// Read the first line
	log, ok := <-ch
	suite.True(ok)
	suite.Equal(models.ExecutionLogTypeSTDOUT, log.Value.Type)
	suite.Equal("stdout line 1\n", log.Value.Line)

	// Read the next line from STDOUT and verify that the channel is closed
	_, ok = <-ch
	suite.False(ok)
}

// TestCompletedStreamerSuite runs the CompletedStreamer test suite
func TestCompletedStreamerSuite(t *testing.T) {
	suite.Run(t, new(CompletedStreamerSuite))
}
