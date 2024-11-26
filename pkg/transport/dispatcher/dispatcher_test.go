//go:build unit || !integration

package dispatcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
)

type DispatcherTestSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	ctx       context.Context
	publisher *ncl.MockOrderedPublisher
	watcher   *watcher.MockWatcher
	creator   *transport.MockMessageCreator
	config    Config
	handler   watcher.EventHandler
}

func (suite *DispatcherTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.publisher = ncl.NewMockOrderedPublisher(suite.ctrl)
	suite.watcher = watcher.NewMockWatcher(suite.ctrl)
	suite.creator = transport.NewMockMessageCreator(suite.ctrl)
	suite.config = DefaultConfig()
}

func (suite *DispatcherTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DispatcherTestSuite) TestNewDispatcher() {
	tests := []struct {
		name        string
		setup       func() (*Dispatcher, error)
		expectError string
	}{
		{
			name: "nil publisher",
			setup: func() (*Dispatcher, error) {
				return New(nil, suite.watcher, suite.creator, suite.config)
			},
			expectError: "publisher cannot be nil",
		},
		{
			name: "nil watcher",
			setup: func() (*Dispatcher, error) {
				return New(suite.publisher, nil, suite.creator, suite.config)
			},
			expectError: "watcher cannot be nil",
		},
		{
			name: "handler setup failure",
			setup: func() (*Dispatcher, error) {
				suite.watcher.EXPECT().SetHandler(gomock.Any()).Return(fmt.Errorf("handler error"))
				return New(suite.publisher, suite.watcher, suite.creator, suite.config)
			},
			expectError: "failed to set handler",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			d, err := tc.setup()
			if tc.expectError != "" {
				suite.Error(err)
				suite.ErrorContains(err, tc.expectError)
				suite.Nil(d)
			} else {
				suite.NoError(err)
				suite.NotNil(d)
			}
		})
	}
}

func (suite *DispatcherTestSuite) TestStartupFailure() {
	suite.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)

	d, err := New(suite.publisher, suite.watcher, suite.creator, suite.config)
	suite.Require().NoError(err)

	startErr := fmt.Errorf("start failed")
	suite.watcher.EXPECT().Start(gomock.Any()).Return(startErr)

	err = d.Start(suite.ctx)
	suite.Error(err)
	suite.ErrorIs(err, startErr)
}

func (suite *DispatcherTestSuite) TestDoubleStart() {
	suite.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)
	suite.watcher.EXPECT().Start(gomock.Any()).Return(nil)

	d, err := New(suite.publisher, suite.watcher, suite.creator, suite.config)
	suite.Require().NoError(err)

	err = d.Start(suite.ctx)
	suite.NoError(err)

	err = d.Start(suite.ctx)
	suite.Error(err)
	suite.Contains(err.Error(), "already running")
}

func (suite *DispatcherTestSuite) TestStopTimeout() {
	suite.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)
	suite.watcher.EXPECT().Start(gomock.Any()).Return(nil)

	d, err := New(suite.publisher, suite.watcher, suite.creator, suite.config)
	suite.Require().NoError(err)

	err = d.Start(suite.ctx)
	suite.NoError(err)

	// Create timeout context
	ctx, cancel := context.WithDeadline(suite.ctx, time.Now().Add(-1*time.Millisecond))
	defer cancel()

	suite.watcher.EXPECT().Stop(gomock.Any())

	err = d.Stop(ctx)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "timed out")
}

func (suite *DispatcherTestSuite) TestDoubleStop() {
	// Setup expectations for creating and starting the dispatcher
	suite.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)
	suite.watcher.EXPECT().Start(gomock.Any()).Return(nil)

	d, err := New(suite.publisher, suite.watcher, suite.creator, suite.config)
	suite.Require().NoError(err)

	// Start the dispatcher
	err = d.Start(suite.ctx)
	suite.NoError(err)

	// First stop should call watcher.Stop
	suite.watcher.EXPECT().Stop(gomock.Any())

	// First stop should succeed
	err = d.Stop(suite.ctx)
	suite.NoError(err)

	// Second stop should return immediately without error and without calling watcher.Stop again
	err = d.Stop(suite.ctx)
	suite.NoError(err)
}

func (suite *DispatcherTestSuite) TestStopNonStarted() {
	// Setup the dispatcher without starting it
	suite.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)

	d, err := New(suite.publisher, suite.watcher, suite.creator, suite.config)
	suite.Require().NoError(err)

	// Stop without start should return immediately without error
	// and without calling watcher.Stop
	err = d.Stop(suite.ctx)
	suite.NoError(err)
}

func TestDispatcherTestSuite(t *testing.T) {
	suite.Run(t, new(DispatcherTestSuite))
}
