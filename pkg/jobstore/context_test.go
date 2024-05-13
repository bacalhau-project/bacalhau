//go:build unit || !integration

package jobstore

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type mockTxContext struct {
	context.Context
}

// newMockTxContext creates a new mock transactional context
func newMockTxContext(ctx context.Context) mockTxContext {
	return mockTxContext{Context: ctx}
}

func (m mockTxContext) Commit() error {
	return nil
}

func (m mockTxContext) Rollback() error {
	return nil
}

type TracingContextTestSuite struct {
	suite.Suite
	originalLogger zerolog.Logger
	buf            zerolog.ConsoleWriter
}

func (s *TracingContextTestSuite) SetupTest() {
	// Capture the original logger
	s.originalLogger = log.Logger

	// Capture logs in a buffer
	s.buf = zerolog.ConsoleWriter{Out: &bytes.Buffer{}, TimeFormat: time.RFC3339}
	log.Logger = log.Output(s.buf)
}

func (s *TracingContextTestSuite) TearDownTest() {
	// Restore the original logger
	log.Logger = s.originalLogger
}

func (s *TracingContextTestSuite) TestCommitAndRollbackWithoutDelay() {
	// Create a new TracingContext
	ctx := NewTracingContext(newMockTxContext(context.Background()))

	// Test Commit method without delay
	err := ctx.Commit()
	assert.NoError(s.T(), err)
	assert.NotContains(s.T(), s.buf.Out.(*bytes.Buffer).String(), "transaction took longer than")

	// Test Rollback method without delay
	err = ctx.Rollback()
	assert.NoError(s.T(), err)
	assert.NotContains(s.T(), s.buf.Out.(*bytes.Buffer).String(), "transaction took longer than")
}

func (s *TracingContextTestSuite) TestCommitAndRollbackWithDelay() {
	// Reset the buffer
	s.buf.Out.(*bytes.Buffer).Reset()

	// Create a new TracingContext with delay
	ctx := NewTracingContext(newMockTxContext(context.Background()))
	time.Sleep(maxTracingDuration + 1*time.Nanosecond) // simulate a delay between creation and commit/rollback

	// Test Commit method with delay
	err := ctx.Commit()
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), s.buf.Out.(*bytes.Buffer).String(), "transaction took longer than")

	// Test Rollback method with delay
	err = ctx.Rollback()
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), s.buf.Out.(*bytes.Buffer).String(), "transaction took longer than")
}

func TestTracingContextTestSuite(t *testing.T) {
	suite.Run(t, new(TracingContextTestSuite))
}
