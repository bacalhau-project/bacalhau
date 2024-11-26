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
	suite.False(suite.recovery.isInRecovery())
	suite.Equal(0, suite.recovery.failures)
	suite.True(suite.recovery.lastFailure.IsZero())

	// Handle error
	suite.recovery.handleError(suite.ctx, msg, publishErr)

	// Verify final state
	suite.Eventually(func() bool {
		return !suite.recovery.isInRecovery() &&
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
	startErr := fmt.Errorf("start failed")

	// Expect multiple recovery attempts
	gomock.InOrder(
		suite.watcher.EXPECT().Start(gomock.Any()).Return(startErr),
		suite.watcher.EXPECT().Stats().Return(watcher.Stats{State: watcher.StateStopped}),
		suite.watcher.EXPECT().Start(gomock.Any()).Return(nil),
	)

	suite.recovery.recoveryLoop(suite.ctx, 1)

	// Verify final state
	recovering, _, failures := suite.recovery.getState()
	suite.False(recovering)
	suite.Equal(0, failures)
}

func (suite *RecoveryTestSuite) TestRecoveryLoopWithRunningWatcher() {
	// Expect watcher already running
	suite.watcher.EXPECT().Start(gomock.Any()).Return(fmt.Errorf("some error"))
	suite.watcher.EXPECT().Stats().Return(watcher.Stats{State: watcher.StateRunning})

	suite.recovery.recoveryLoop(suite.ctx, 1)

	// Verify final state
	recovering, _, failures := suite.recovery.getState()
	suite.False(recovering)
	suite.Equal(0, failures)
}

func (suite *RecoveryTestSuite) TestRecoveryLoopWithContextCancellation() {
	ctx, cancel := context.WithCancel(suite.ctx)
	cancel()

	// Expect one failed attempt followed by context cancellation
	suite.watcher.EXPECT().Start(gomock.Any()).Return(fmt.Errorf("start error"))
	suite.watcher.EXPECT().Stats().Return(watcher.Stats{State: watcher.StateStopped})

	suite.recovery.recoveryLoop(ctx, 1)

	// Verify final state
	recovering, _, failures := suite.recovery.getState()
	suite.False(recovering)
	suite.Equal(0, failures)
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

func TestRecoveryTestSuite(t *testing.T) {
	suite.Run(t, new(RecoveryTestSuite))
}
