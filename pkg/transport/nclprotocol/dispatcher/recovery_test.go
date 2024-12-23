//go:build unit || !integration

package dispatcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

type RecoveryTestSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	ctx       context.Context
	publisher *ncl.MockOrderedPublisher
	watcher   *watcher.MockWatcher
	state     *dispatcherState
	recovery  *recovery
}

func (suite *RecoveryTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.publisher = ncl.NewMockOrderedPublisher(suite.ctrl)
	suite.watcher = watcher.NewMockWatcher(suite.ctrl)
	suite.state = newDispatcherState()
	suite.recovery = newRecovery(
		suite.publisher,
		suite.watcher,
		suite.state,
		Config{
			BaseRetryInterval: 50 * time.Millisecond,
			MaxRetryInterval:  200 * time.Millisecond,
		},
	)
}

func (suite *RecoveryTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *RecoveryTestSuite) TestHandleFirstError() {
	future := ncl.NewMockPubFuture(suite.ctrl)
	future.EXPECT().Msg().Return(&nats.Msg{
		Data: []byte("test"),
	}).AnyTimes()

	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      future,
	}
	publishErr := fmt.Errorf("publish failed")

	// Set up mock expectations in sequence
	gomock.InOrder(
		// Recovery sequence
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
		suite.watcher.EXPECT().Start(gomock.Any()).Return(nil),
	)

	// Initial state checks
	suite.False(suite.recovery.isRecovering)
	suite.Equal(0, suite.recovery.failures)
	suite.True(suite.recovery.lastFailure.IsZero())

	// Handle error
	suite.recovery.handleError(suite.ctx, msg, publishErr)

	// Verify final state
	suite.Eventually(func() bool {
		return !suite.recovery.isRecovering &&
			suite.recovery.failures == 1
	}, 1*time.Second, 50*time.Millisecond)
}

func (suite *RecoveryTestSuite) TestHandleErrorWhileRecovering() {
	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      ncl.NewMockPubFuture(suite.ctrl),
	}
	publishErr := fmt.Errorf("publish failed")

	// Simulate already in recovery
	suite.recovery.mu.Lock()
	suite.recovery.isRecovering = true
	suite.recovery.failures = 1
	suite.recovery.lastFailure = time.Now()
	suite.recovery.mu.Unlock()

	// No expectations - should not trigger recovery sequence

	// Handle error
	suite.recovery.handleError(suite.ctx, msg, publishErr)

	// Verify state unchanged
	recovering, _, failures := suite.recovery.getState()
	suite.True(recovering)
	suite.Equal(1, failures)
}

func (suite *RecoveryTestSuite) TestRecoveryLoopWithFailures() {
	// Setup mock future for handleError
	future := ncl.NewMockPubFuture(suite.ctrl)
	future.EXPECT().Msg().Return(&nats.Msg{Data: []byte("test")}).AnyTimes()

	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      future,
	}

	// Expect recovery sequence
	gomock.InOrder(
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
		suite.watcher.EXPECT().Start(gomock.Any()).Return(fmt.Errorf("start failed")),
		suite.watcher.EXPECT().Stats().Return(watcher.Stats{State: watcher.StateStopped}),
		suite.watcher.EXPECT().Start(gomock.Any()).Return(nil),
	)

	// Start recovery properly through handleError
	suite.recovery.handleError(suite.ctx, msg, fmt.Errorf("test error"))

	// Wait for recovery to complete
	suite.Eventually(func() bool {
		recovering, _, _ := suite.recovery.getState()
		return !recovering
	}, time.Second, 10*time.Millisecond)
}

func (suite *RecoveryTestSuite) TestRecoveryLoopWithRunningWatcher() {
	// Setup mock future for handleError
	future := ncl.NewMockPubFuture(suite.ctrl)
	future.EXPECT().Msg().Return(&nats.Msg{Data: []byte("test")}).AnyTimes()

	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      future,
	}

	// Expect recovery sequence
	gomock.InOrder(
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
		suite.watcher.EXPECT().Start(gomock.Any()).Return(fmt.Errorf("some error")),
		suite.watcher.EXPECT().Stats().Return(watcher.Stats{State: watcher.StateRunning}),
	)

	suite.recovery.handleError(suite.ctx, msg, fmt.Errorf("test error"))

	suite.Eventually(func() bool {
		recovering, _, _ := suite.recovery.getState()
		return !recovering
	}, time.Second, 10*time.Millisecond)
}

func (suite *RecoveryTestSuite) TestRecoveryLoopWithContextCancellation() {
	// Create cancellable context
	ctx, cancel := context.WithCancel(suite.ctx)

	// Setup mock future
	future := ncl.NewMockPubFuture(suite.ctrl)
	future.EXPECT().Msg().Return(&nats.Msg{Data: []byte("test")}).AnyTimes()

	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      future,
	}

	// Expect initial recovery sequence
	gomock.InOrder(
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
	)

	// Start recovery
	suite.recovery.handleError(ctx, msg, fmt.Errorf("test error"))

	// Cancel context during backoff
	cancel()

	// Verify recovery eventually completes
	suite.Eventually(func() bool {
		recovering, _, _ := suite.recovery.getState()
		return !recovering
	}, time.Second, 10*time.Millisecond)
}

func (suite *RecoveryTestSuite) TestReset() {
	// Setup initial state
	suite.recovery.mu.Lock()
	suite.recovery.isRecovering = true
	suite.recovery.failures = 5
	suite.recovery.lastFailure = time.Now()
	suite.recovery.mu.Unlock()

	// Reset
	suite.recovery.reset()

	// Verify all state cleared
	recovering, lastFailure, failures := suite.recovery.getState()
	suite.False(recovering)
	suite.True(lastFailure.IsZero())
	suite.Equal(0, failures)
}

func (suite *RecoveryTestSuite) TestStopDuringRecovery() {
	// Setup mock future
	future := ncl.NewMockPubFuture(suite.ctrl)
	future.EXPECT().Msg().Return(&nats.Msg{Data: []byte("test")}).AnyTimes()

	// Setup recovery sequence
	gomock.InOrder(
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
		// Don't expect watcher.Start since we'll stop before that
	)

	// Start recovery
	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      future,
	}
	suite.recovery.handleError(suite.ctx, msg, fmt.Errorf("publish failed"))

	// Verify recovery started
	recovering, _, _ := suite.recovery.getState()
	suite.True(recovering)

	// Stop recovery - should block until loop exits
	done := make(chan struct{})
	go func() {
		suite.recovery.stop()
		close(done)
	}()

	// Should complete quickly since we interrupt backoff
	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		suite.Fail("Recovery stop did not complete in time")
	}

	// Verify recovery cleaned up
	recovering, _, _ = suite.recovery.getState()
	suite.False(recovering)
}

func (suite *RecoveryTestSuite) TestStopAfterRecoveryComplete() {
	// Setup mock future and complete recovery sequence
	future := ncl.NewMockPubFuture(suite.ctrl)
	future.EXPECT().Msg().Return(&nats.Msg{Data: []byte("test")}).AnyTimes()

	gomock.InOrder(
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
		suite.watcher.EXPECT().Start(gomock.Any()).Return(nil),
	)

	// Run recovery to completion
	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      future,
	}
	suite.recovery.handleError(suite.ctx, msg, fmt.Errorf("publish failed"))

	// Wait for recovery to complete
	suite.Eventually(func() bool {
		recovering, _, _ := suite.recovery.getState()
		return !recovering
	}, time.Second, 10*time.Millisecond)

	// Stop should complete immediately since no recovery is running
	done := make(chan struct{})
	go func() {
		suite.recovery.stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		suite.Fail("Stop took too long when no recovery was running")
	}
}

func (suite *RecoveryTestSuite) TestMultipleStopCalls() {
	// First stop should complete normally
	suite.recovery.stop()

	// Additional stops should complete immediately
	done := make(chan struct{})
	go func() {
		suite.recovery.stop()
		suite.recovery.stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		suite.Fail("Multiple stops took too long")
	}
}

func (suite *RecoveryTestSuite) TestStopAndReset() {
	future := ncl.NewMockPubFuture(suite.ctrl)
	future.EXPECT().Msg().Return(&nats.Msg{Data: []byte("test")}).AnyTimes()

	msg := &pendingMessage{
		eventSeqNum: 1,
		publishTime: time.Now(),
		future:      future,
	}

	// First recovery sequence
	gomock.InOrder(
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
	)

	// Second recovery sequence after reset
	gomock.InOrder(
		suite.watcher.EXPECT().Stop(gomock.Any()),
		suite.publisher.EXPECT().Reset(gomock.Any()),
	)

	suite.recovery.handleError(suite.ctx, msg, fmt.Errorf("test error"))
	suite.recovery.stop()
	suite.recovery.reset()
	suite.recovery.handleError(suite.ctx, msg, fmt.Errorf("test error"))
	suite.recovery.stop()
}

func TestRecoveryTestSuite(t *testing.T) {
	suite.Run(t, new(RecoveryTestSuite))
}
