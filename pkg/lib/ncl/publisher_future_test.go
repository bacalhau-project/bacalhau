//go:build unit || !integration

package ncl

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

type PubFutureTestSuite struct {
	suite.Suite
}

func (suite *PubFutureTestSuite) TestNewPubFuture() {
	msg := &nats.Msg{Subject: "test"}
	future := newPubFuture(msg)

	suite.NotNil(future)
	suite.Equal(msg, future.Msg())
	suite.Nil(future.Result())
	suite.Nil(future.Err())
	suite.NotNil(future.Done())
}

func (suite *PubFutureTestSuite) TestSetResult() {
	future := newPubFuture(&nats.Msg{})
	result := NewResult().WithDelay(5 * time.Second)

	// Set result
	future.setResult(result)

	// Verify result is set
	suite.Equal(result, future.Result())
	suite.Nil(future.Err())
	suite.Equal(5*time.Second, future.Result().DelayDuration())

	// Verify Done channel is closed
	select {
	case <-future.Done():
		// Success
	default:
		suite.Fail("Done channel should be closed")
	}

	// Verify subsequent setResult has no effect
	newResult := NewResult().WithDelay(10 * time.Second)
	future.setResult(newResult)
	suite.Equal(result, future.Result(), "Result should not change after first set")
	suite.Equal(5*time.Second, future.Result().DelayDuration())
}

func (suite *PubFutureTestSuite) TestSetError() {
	future := newPubFuture(&nats.Msg{})
	testErr := errors.New("test error")

	// Set error
	future.setError(testErr)

	// Verify error is set
	suite.Equal(testErr, future.Err())
	suite.Nil(future.Result())

	// Verify Done channel is closed
	select {
	case <-future.Done():
		// Success
	default:
		suite.Fail("Done channel should be closed")
	}

	// Verify subsequent setError has no effect
	newErr := errors.New("new error")
	future.setError(newErr)
	suite.Equal(testErr, future.Err(), "Error should not change after first set")
}

func (suite *PubFutureTestSuite) TestWait() {
	future := newPubFuture(&nats.Msg{})

	// Test wait with completion
	go func() {
		time.Sleep(50 * time.Millisecond)
		future.setResult(NewResult().WithDelay(time.Second))
	}()

	ctx := context.Background()
	err := future.Wait(ctx)
	suite.NoError(err)
	suite.NotNil(future.Result())
	suite.Equal(time.Second, future.Result().DelayDuration())

	// Test wait with context cancellation
	future = newPubFuture(&nats.Msg{})
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err = future.Wait(ctx)
	suite.Error(err)
	suite.True(errors.Is(err, context.Canceled))
}

func (suite *PubFutureTestSuite) TestWaitTimeout() {
	future := newPubFuture(&nats.Msg{})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Wait should return error after timeout
	err := future.Wait(ctx)
	suite.Error(err)
	suite.True(errors.Is(err, context.DeadlineExceeded))
}

func (suite *PubFutureTestSuite) TestConcurrentAccess() {
	future := newPubFuture(&nats.Msg{})
	result := NewResult().WithDelay(time.Second)
	errResult := errors.New("test error")

	// Launch multiple goroutines trying to set result and error
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			future.setResult(result)
		}()
		go func() {
			defer wg.Done()
			future.setError(errResult)
		}()
	}

	wg.Wait()

	// Verify only one operation succeeded
	suite.True((future.Result() != nil && future.Err() == nil) ||
		(future.Result() == nil && future.Err() != nil))

	// If result was set, verify it has expected delay
	if future.Result() != nil {
		suite.Equal(time.Second, future.Result().DelayDuration())
	}

	// Verify Done channel is closed exactly once
	select {
	case <-future.Done():
		// Try reading again to verify channel wasn't closed multiple times
		select {
		case _, ok := <-future.Done():
			suite.False(ok, "Done channel should remain closed")
		default:
			suite.Fail("Done channel should be closed")
		}
	default:
		suite.Fail("Done channel should be closed")
	}
}

func TestPubFutureTestSuite(t *testing.T) {
	suite.Run(t, new(PubFutureTestSuite))
}
