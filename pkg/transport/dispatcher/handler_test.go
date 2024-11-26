//go:build unit || !integration

package dispatcher

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
)

type HandlerTestSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	ctx       context.Context
	creator   *transport.MockMessageCreator
	publisher *ncl.MockOrderedPublisher
	state     *dispatcherState
	handler   *messageHandler
}

func (suite *HandlerTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.creator = transport.NewMockMessageCreator(suite.ctrl)
	suite.publisher = ncl.NewMockOrderedPublisher(suite.ctrl)
	suite.state = newDispatcherState()
	suite.handler = newMessageHandler(suite.creator, suite.publisher, suite.state)
}

func (suite *HandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *HandlerTestSuite) TestHandleEventCreatorError() {
	event := watcher.Event{SeqNum: 1}
	expectedErr := fmt.Errorf("creation failed")

	suite.creator.EXPECT().
		CreateMessage(event).
		Return(nil, expectedErr)

	err := suite.handler.HandleEvent(suite.ctx, event)
	suite.Error(err)
	suite.ErrorContains(err, expectedErr.Error())
	suite.Equal(uint64(0), suite.state.lastObservedSeq)
}

func (suite *HandlerTestSuite) TestHandleEventNilMessage() {
	event := watcher.Event{SeqNum: 1}

	suite.creator.EXPECT().
		CreateMessage(event).
		Return(nil, nil)

	err := suite.handler.HandleEvent(suite.ctx, event)
	suite.NoError(err)
	suite.Equal(event.SeqNum, suite.state.lastObservedSeq)
	suite.Equal(0, suite.state.pending.Size())
}

func (suite *HandlerTestSuite) TestHandleEventPublishError() {
	event := watcher.Event{SeqNum: 1}
	msg := envelope.NewMessage("msg")
	expectedErr := fmt.Errorf("publish failed")

	suite.creator.EXPECT().
		CreateMessage(event).
		Return(msg, nil)

	suite.publisher.EXPECT().
		PublishAsync(suite.ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, req ncl.PublishRequest) (ncl.PubFuture, error) {
			// Verify message enrichment
			suite.Equal(msg, req.Message)
			suite.Equal(fmt.Sprint(event.SeqNum), req.Message.Metadata.Get(KeySeqNum))
			suite.Equal(generateMsgID(event), req.Message.Metadata.Get(ncl.KeyMessageID))
			return nil, expectedErr
		})

	err := suite.handler.HandleEvent(suite.ctx, event)
	suite.Error(err)
	suite.ErrorContains(err, expectedErr.Error())
	suite.Equal(uint64(0), suite.state.lastObservedSeq)
	suite.Equal(0, suite.state.pending.Size())
}

func (suite *HandlerTestSuite) TestHandleEventSuccess() {
	event := watcher.Event{SeqNum: 1}
	msg := envelope.NewMessage("msg").WithMetadataValue(ncl.KeySubject, "test.subject")
	future := ncl.NewMockPubFuture(suite.ctrl)

	suite.creator.EXPECT().
		CreateMessage(event).
		Return(msg, nil)

	suite.publisher.EXPECT().
		PublishAsync(suite.ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, req ncl.PublishRequest) (ncl.PubFuture, error) {
			// Verify message enrichment
			suite.Equal("test.subject", req.Subject)
			suite.Equal(fmt.Sprint(event.SeqNum), req.Message.Metadata.Get(KeySeqNum))
			suite.Equal(generateMsgID(event), req.Message.Metadata.Get(ncl.KeyMessageID))
			return future, nil
		})

	err := suite.handler.HandleEvent(suite.ctx, event)
	suite.NoError(err)
	suite.Equal(event.SeqNum, suite.state.lastObservedSeq)
	suite.Equal(1, suite.state.pending.Size())

	// Verify pending message
	pending := suite.state.pending.GetAll()[0]
	suite.Equal(event.SeqNum, pending.eventSeqNum)
	suite.NotZero(pending.publishTime)
	suite.Equal(future, pending.future)
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
