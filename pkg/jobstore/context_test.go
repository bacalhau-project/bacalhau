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
	buf            bytes.Buffer
	logCtx         context.Context
}

func (s *TracingContextTestSuite) SetupTest() {
	// Capture the original logger
	s.originalLogger = log.Logger

	// Capture logs in a buffer
	log.Logger = zerolog.New(&s.buf).With().Timestamp().Logger()
	s.logCtx = log.Logger.WithContext(context.Background())
}

func (s *TracingContextTestSuite) TearDownTest() {
	// Restore the original logger
	log.Logger = s.originalLogger
}

func (s *TracingContextTestSuite) TestCommitAndRollbackWithoutDelay() {
	// Create a new TracingContext
	ctx := NewTracingContext(newMockTxContext(s.logCtx))

	// Test Commit method without delay
	err := ctx.Commit()
	assert.NoError(s.T(), err)
	assert.NotContains(s.T(), s.buf.String(), "transaction took longer than")

	// Test Rollback method without delay
	err = ctx.Rollback()
	assert.NoError(s.T(), err)
	assert.NotContains(s.T(), s.buf.String(), "transaction took longer than")
}

func (s *TracingContextTestSuite) TestCommitAndRollbackWithDelay() {
	// Create a new TracingContext with delay
	ctx := NewTracingContext(newMockTxContext(s.logCtx))
	time.Sleep(maxTracingDuration + 1*time.Nanosecond) // simulate a delay between creation and commit/rollback

	// Test Commit method with delay
	err := ctx.Commit()
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), s.buf.String(), "transaction took")
	assert.Contains(s.T(), s.buf.String(), "to commit")

	// Test Rollback method with delay
	err = ctx.Rollback()
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), s.buf.String(), "transaction took")
	assert.Contains(s.T(), s.buf.String(), "to rollback")
}

func TestTracingContextTestSuite(t *testing.T) {
	suite.Run(t, new(TracingContextTestSuite))
}
