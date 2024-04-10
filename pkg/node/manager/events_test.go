//go:build unit || !integration

package manager_test

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/stretchr/testify/suite"
)

type EventEmitterSuite struct {
	suite.Suite
	ctx   context.Context
	clock clock.Clock
}

func TestEventEmitterSuite(t *testing.T) {
	suite.Run(t, new(EventEmitterSuite))
}

func (s *EventEmitterSuite) SetupTest() {
	s.ctx = context.Background()
	s.clock = clock.NewMock()
}

func (s *EventEmitterSuite) TestNewNodeEventEmitter() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	e.RegisterCallback(manager.NodeEventApproved,
		func(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
			s.Equal(models.NodeInfo{}, info)
			s.Equal(manager.NodeEventApproved, event)
		},
	)

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventApproved)
	s.NoError(err)
}

func (s *EventEmitterSuite) TestRegisterCallback() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	e.RegisterCallback(manager.NodeEventApproved,
		func(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
			s.Equal(models.NodeInfo{}, info)
			s.Equal(manager.NodeEventApproved, event)
		},
	)

	e.RegisterCallback(manager.NodeEventRejected,
		func(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
			s.Equal(models.NodeInfo{}, info)
			s.Equal(manager.NodeEventRejected, event)
		},
	)
}

func (s *EventEmitterSuite) TestEmitEvent() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	e.RegisterCallback(manager.NodeEventApproved,
		func(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
			s.Equal(models.NodeInfo{}, info)
			s.Equal(manager.NodeEventApproved, event)
		},
	)

	e.RegisterCallback(manager.NodeEventRejected,
		func(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
			s.Equal(models.NodeInfo{}, info)
			s.Equal(manager.NodeEventRejected, event)
		},
	)

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

func (s *EventEmitterSuite) TestEmitEventWithNoCallbacksForEvent() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	e.RegisterCallback(manager.NodeEventRejected,
		func(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
			s.Equal(models.NodeInfo{}, info)
			s.Equal(manager.NodeEventRejected, event)
		},
	)

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventApproved)
	s.NoError(err)
}

func (s *EventEmitterSuite) TestEmitEventWithNoCallbacksForEventAndNoCallbacks() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventApproved)
	s.NoError(err)
}

func (s *EventEmitterSuite) TestEmitWithSlowCallback() {
	e := manager.NewNodeEventEmitter()
	s.NotNil(e)

	e.RegisterCallback(manager.NodeEventRejected,
		func(ctx context.Context, info models.NodeInfo, event manager.NodeEvent) {
			s.clock.Sleep(2 * time.Second)
		},
	)

	err := e.EmitEvent(s.ctx, models.NodeInfo{}, manager.NodeEventRejected)
	s.Error(err)
}
