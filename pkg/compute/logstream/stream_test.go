//go:build unit || !integration

package logstream

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

type LogStreamTestSuite struct {
	suite.Suite
}

func TestLogStreamTestSuite(t *testing.T) {
	suite.Run(t, new(LogStreamTestSuite))
}

type simulatedLogEntry struct {
	Type models.ExecutionLogType
	Line string
}

// Helper function to simulate log stream
func (suite *LogStreamTestSuite) simulateLogStream(logEntries []simulatedLogEntry) []byte {
	var data []byte
	for _, entry := range logEntries {
		header := make([]byte, logger.HeaderLength)
		if entry.Type == models.ExecutionLogTypeSTDOUT {
			header[0] = byte(logger.StdoutStreamTag)
		}
		binary.BigEndian.PutUint32(header[4:], uint32(len(entry.Line)))
		data = append(data, header...)
		data = append(data, []byte(entry.Line)...)
	}
	return data
}

func (suite *LogStreamTestSuite) TestLogStream_MultipleEntries() {
	logEntries := []simulatedLogEntry{
		{Type: models.ExecutionLogTypeSTDOUT, Line: "First line"},
		{Type: models.ExecutionLogTypeSTDERR, Line: "Second line"},
		{Type: models.ExecutionLogTypeSTDOUT, Line: "Third line"},
	}

	data := suite.simulateLogStream(logEntries)
	r := bytes.NewReader(data)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logStream := NewStream(ctx, StreamParams{
		Reader: r,
		Buffer: 10,
	})

	for i, expected := range logEntries {
		log, more := <-logStream.LogChannel
		suite.True(more, "Stream channel closed unexpectedly after %d entries", i)
		suite.Equal(expected.Type, log.Type, "Mismatch in log type for entry %d", i)
		suite.Equal(expected.Line, log.Line, "Mismatch in log line for entry %d", i)
	}

	select {
	case _, more := <-logStream.LogChannel:
		if more {
			suite.Fail("Stream channel should be closed after reading all entries")
		}
	case <-time.After(time.Second):
		suite.Fail("Timeout waiting for Stream channel to close")
	}
}

func (suite *LogStreamTestSuite) TestLogStream_EmptyStream() {
	r := bytes.NewReader([]byte{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logStream := NewStream(ctx, StreamParams{
		Reader: r,
		Buffer: 10,
	})

	select {
	case _, more := <-logStream.LogChannel:
		if more {
			suite.Fail("Stream channel should be closed for an empty stream")
		}
	case <-time.After(time.Second):
		suite.Fail("Timeout waiting for Stream channel to close on empty stream")
	}
}

func (suite *LogStreamTestSuite) TestLogStream_CancelContext() {
	// Simulate long log stream
	logEntries := []simulatedLogEntry{
		{Type: models.ExecutionLogTypeSTDOUT, Line: "Line1"},
	}

	data := suite.simulateLogStream(logEntries)
	r := bytes.NewReader(data)
	ctx, cancel := context.WithCancel(context.Background())

	logStream := NewStream(ctx, StreamParams{
		Reader: r,
		Buffer: 10,
	})

	// Cancel the context
	cancel()

	// Check if the channel is closed after context cancellation
	select {
	case _, more := <-logStream.LogChannel:
		if more {
			suite.Fail("Stream channel should be closed after context cancellation")
		}
	case <-time.After(time.Second * 2):
		suite.Fail("Timeout waiting for Stream channel to close after context cancellation")
	}
}
