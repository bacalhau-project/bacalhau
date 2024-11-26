//go:build unit || !integration

package dispatcher

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
)

type StateTestSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	state   *dispatcherState
	pending *pendingMessageStore
}

func (suite *StateTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.state = newDispatcherState()
	suite.pending = newPendingMessageStore()
}

func (suite *StateTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// Dispatcher State Tests
func (suite *StateTestSuite) TestUpdateLastAcked() {
	// Should take highest value
	suite.state.updateLastAcked(5)
	suite.state.updateLastAcked(3)
	suite.state.updateLastAcked(7)

	suite.Equal(uint64(7), suite.state.lastAckedSeqNum)
}

func (suite *StateTestSuite) TestUpdateLastObserved() {
	// Should take latest value
	suite.state.updateLastObserved(5)
	suite.state.updateLastObserved(3)
	suite.state.updateLastObserved(7)

	suite.Equal(uint64(7), suite.state.lastObservedSeq)
}

func (suite *StateTestSuite) TestGetCheckpointSeqNum() {
	testCases := []struct {
		name           string
		setup          func()
		expectedSeqNum uint64
	}{
		{
			name: "no pending messages",
			setup: func() {
				suite.state.updateLastObserved(10)
				suite.state.updateLastAcked(5)
			},
			expectedSeqNum: 10, // should use lastObserved
		},
		{
			name: "with pending messages",
			setup: func() {
				suite.state.updateLastObserved(10)
				suite.state.updateLastAcked(5)
				future := ncl.NewMockPubFuture(suite.ctrl)
				suite.state.pending.Add(&pendingMessage{eventSeqNum: 7, future: future})
			},
			expectedSeqNum: 5, // should use lastAcked
		},
		{
			name: "no new checkpoint needed",
			setup: func() {
				suite.state.updateLastObserved(10)
				suite.state.updateLastAcked(5)
				suite.state.updateLastCheckpoint(10)
			},
			expectedSeqNum: 0, // no new checkpoint needed
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.setup()
			suite.Equal(tc.expectedSeqNum, suite.state.getCheckpointSeqNum())
		})
	}
}

func (suite *StateTestSuite) TestReset() {
	// Setup initial state
	suite.state.updateLastAcked(5)
	suite.state.updateLastObserved(10)
	suite.state.updateLastCheckpoint(3)
	future := ncl.NewMockPubFuture(suite.ctrl)
	suite.state.pending.Add(&pendingMessage{eventSeqNum: 7, future: future})

	// Reset
	suite.state.reset()

	// Verify all state cleared
	suite.Equal(uint64(0), suite.state.lastAckedSeqNum)
	suite.Equal(uint64(0), suite.state.lastObservedSeq)
	suite.Equal(uint64(0), suite.state.lastCheckpoint)
	suite.Equal(0, suite.state.pending.Size())
}

// Pending Message Store Tests
func (suite *StateTestSuite) TestPendingMessageStoreAdd() {
	future := ncl.NewMockPubFuture(suite.ctrl)
	msg := &pendingMessage{eventSeqNum: 1, publishTime: time.Now(), future: future}

	suite.pending.Add(msg)
	suite.Equal(1, suite.pending.Size())
	suite.Equal(msg, suite.pending.GetAll()[0])
}

func (suite *StateTestSuite) TestPendingMessageStoreRemoveUpTo() {
	futures := make([]ncl.PubFuture, 5)
	for i := range futures {
		future := ncl.NewMockPubFuture(suite.ctrl)
		future.EXPECT().Msg().Return(&nats.Msg{}).AnyTimes()
		futures[i] = future
		suite.pending.Add(&pendingMessage{
			eventSeqNum: uint64(i + 1),
			publishTime: time.Now(),
			future:      futures[i],
		})
	}

	testCases := []struct {
		name          string
		removeUpTo    uint64
		expectedSize  int
		expectedFirst uint64
	}{
		{
			name:          "remove none",
			removeUpTo:    0,
			expectedSize:  5,
			expectedFirst: 1,
		},
		{
			name:          "remove some",
			removeUpTo:    3,
			expectedSize:  2,
			expectedFirst: 4,
		},
		{
			name:          "remove all",
			removeUpTo:    5,
			expectedSize:  0,
			expectedFirst: 0,
		},
		{
			name:          "remove beyond",
			removeUpTo:    10,
			expectedSize:  0,
			expectedFirst: 0,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			store := newPendingMessageStore()
			for i := range futures {
				store.Add(&pendingMessage{
					eventSeqNum: uint64(i + 1),
					publishTime: time.Now(),
					future:      futures[i],
				})
			}

			store.RemoveUpTo(tc.removeUpTo)
			suite.Equal(tc.expectedSize, store.Size())

			msgs := store.GetAll()
			if tc.expectedSize > 0 {
				suite.Equal(tc.expectedFirst, msgs[0].eventSeqNum)
			}
		})
	}
}

func (suite *StateTestSuite) TestPendingMessageStoreConcurrency() {
	// Test concurrent access
	const goroutines = 10
	const messagesPerRoutine = 100

	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func(base int) {
			for j := 0; j < messagesPerRoutine; j++ {
				future := ncl.NewMockPubFuture(suite.ctrl)
				suite.pending.Add(&pendingMessage{
					eventSeqNum: uint64(base*messagesPerRoutine + j),
					future:      future,
				})
			}
			done <- struct{}{}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	suite.Equal(goroutines*messagesPerRoutine, suite.pending.Size())
}

func TestStateTestSuite(t *testing.T) {
	suite.Run(t, new(StateTestSuite))
}
