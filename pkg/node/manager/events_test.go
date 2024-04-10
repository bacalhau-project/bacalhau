//go:build unit || !integration

package manager_test

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	gomock "go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/stretchr/testify/suite"
)

type EventEmitterSuite struct {
	suite.Suite
	ctrl  *gomock.Controller
	ctx   context.Context
	clock *clock.Mock
}

func TestEventEmitterSuite(t *testing.T) {
	suite.Run(t, new(EventEmitterSuite))
}

func (s *EventEmitterSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.clock = clock.NewMock()
}

func (s *EventEmitterSuite) TestNewNodeEventEmitter() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	mockHandler := manager.NewMockNodeEventHandler(s.ctrl)
	mockHandler.EXPECT().HandleNodeEvent(gomock.Any(), gomock.Any(), manager.NodeEventApproved)

	e.RegisterHandler(mockHandler)

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventApproved)
	s.NoError(err)
}

func (s *EventEmitterSuite) TestRegisterCallback() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	mockHandler := manager.NewMockNodeEventHandler(s.ctrl)
	e.RegisterHandler(mockHandler)
}

func (s *EventEmitterSuite) TestEmitEvent() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	mockHandler := manager.NewMockNodeEventHandler(s.ctrl)
	mockHandler.EXPECT().HandleNodeEvent(gomock.Any(), gomock.Any(), manager.NodeEventApproved)
	mockHandler.EXPECT().HandleNodeEvent(gomock.Any(), gomock.Any(), manager.NodeEventRejected)

	e.RegisterHandler(mockHandler)

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventApproved)
	s.NoError(err)

	err = e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventRejected)
	s.NoError(err)
}

func (s *EventEmitterSuite) TestEmitEventWithNoCallbacks() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventApproved)
	s.NoError(err)
}

func (s *EventEmitterSuite) TestEmitWithSlowCallback() {
	e := manager.NewNodeEventEmitter(manager.WithClock(s.clock))
	s.NotNil(e)

	e.RegisterHandler(testSleepyHandler{s.clock})

	go func() {
		s.clock.Add(10 * time.Second)
	}()

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventRejected)
	s.Error(err)
}

type testSleepyHandler struct {
	c *clock.Mock
}

func (t testSleepyHandler) HandleNodeEvent(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
	t.c.Sleep(2 * time.Second)
}
