//go:build unit || !integration

package backoff

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ExponentialBackoffSuite struct {
	suite.Suite
	baseBackoff time.Duration
	maxBackoff  time.Duration
	delta       time.Duration
	backoff     *Exponential
}

func (s *ExponentialBackoffSuite) SetupTest() {
	s.delta = 20 * time.Millisecond
	s.baseBackoff = 50 * time.Millisecond
	s.maxBackoff = 500 * time.Millisecond
	s.backoff = NewExponential(s.baseBackoff, s.maxBackoff)
}

func (s *ExponentialBackoffSuite) TestBackoffDurationElapsed() {
	testCases := []struct {
		name     string
		attempts int
		expected time.Duration
	}{
		{
			name:     "Attempts0",
			attempts: 0,
			expected: 0,
		},
		{
			name:     "Attempts1",
			attempts: 1,
			expected: s.baseBackoff,
		},
		{
			name:     "Attempts2",
			attempts: 2,
			expected: 2 * s.baseBackoff,
		},
		{
			name:     "Attempts4",
			attempts: 4,
			expected: 8 * s.baseBackoff,
		},
		{
			name:     "Attempts100",
			attempts: 100,
			expected: s.maxBackoff,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := context.Background()
			startTime := time.Now()
			s.backoff.Backoff(ctx, tc.attempts)
			elapsedTime := time.Since(startTime)
			s.InDelta(float64(tc.expected), float64(elapsedTime), float64(s.delta), "Attempt %d: Expected backoff duration of %v, got %v", tc.attempts, tc.expected, elapsedTime)
		})
	}
}

func (s *ExponentialBackoffSuite) TestContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	startTime := time.Now()
	attempts := 2
	go func() {
		time.Sleep(s.baseBackoff / 2)
		cancel()
	}()
	s.backoff.Backoff(ctx, attempts)
	elapsedTime := time.Since(startTime)
	s.InDelta(float64(s.baseBackoff/2), float64(elapsedTime), float64(s.delta))
}

func (s *ExponentialBackoffSuite) TestContextCancelledImmediately() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	startTime := time.Now()
	attempts := 3
	s.backoff.Backoff(ctx, attempts)
	elapsedTime := time.Since(startTime)
	s.InDelta(time.Duration(0), float64(elapsedTime), float64(s.delta))
}

func TestExponentialBackoffSuite(t *testing.T) {
	suite.Run(t, new(ExponentialBackoffSuite))
}
