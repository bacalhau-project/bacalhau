package forwarder

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
)

type ForwarderUnitTestSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	ctx       context.Context
	publisher *ncl.MockOrderedPublisher
	watcher   *watcher.MockWatcher
	creator   *transport.MockMessageCreator
}

func (s *ForwarderUnitTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.publisher = ncl.NewMockOrderedPublisher(s.ctrl)
	s.watcher = watcher.NewMockWatcher(s.ctrl)
	s.creator = transport.NewMockMessageCreator(s.ctrl)
}

func (s *ForwarderUnitTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *ForwarderUnitTestSuite) TestNewForwarder() {
	tests := []struct {
		name        string
		setup       func() (*Forwarder, error)
		expectError string
	}{
		{
			name: "nil publisher",
			setup: func() (*Forwarder, error) {
				return New(nil, s.watcher, s.creator)
			},
			expectError: "publisher cannot be nil",
		},
		{
			name: "nil watcher",
			setup: func() (*Forwarder, error) {
				return New(s.publisher, nil, s.creator)
			},
			expectError: "watcher cannot be nil",
		},
		{
			name: "nil message creator",
			setup: func() (*Forwarder, error) {
				return New(s.publisher, s.watcher, nil)
			},
			expectError: "message creator cannot be nil",
		},
		{
			name: "handler setup failure",
			setup: func() (*Forwarder, error) {
				s.watcher.EXPECT().SetHandler(gomock.Any()).Return(fmt.Errorf("handler error"))
				return New(s.publisher, s.watcher, s.creator)
			},
			expectError: "failed to set handler",
		},
		{
			name: "success",
			setup: func() (*Forwarder, error) {
				s.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)
				return New(s.publisher, s.watcher, s.creator)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			f, err := tc.setup()
			if tc.expectError != "" {
				s.Error(err)
				s.ErrorContains(err, tc.expectError)
				s.Nil(f)
			} else {
				s.NoError(err)
				s.NotNil(f)
			}
		})
	}
}

func (s *ForwarderUnitTestSuite) TestStartupFailure() {
	s.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)

	f, err := New(s.publisher, s.watcher, s.creator)
	s.Require().NoError(err)

	startErr := fmt.Errorf("start failed")
	s.watcher.EXPECT().Start(gomock.Any()).Return(startErr)

	err = f.Start(s.ctx)
	s.Error(err)
	s.ErrorIs(err, startErr)
}

func (s *ForwarderUnitTestSuite) TestDoubleStart() {
	s.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)
	s.watcher.EXPECT().Start(gomock.Any()).Return(nil)

	f, err := New(s.publisher, s.watcher, s.creator)
	s.Require().NoError(err)

	err = f.Start(s.ctx)
	s.NoError(err)

	err = f.Start(s.ctx)
	s.Error(err)
	s.Contains(err.Error(), "already running")
}

func (s *ForwarderUnitTestSuite) TestStopNonStarted() {
	s.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)

	f, err := New(s.publisher, s.watcher, s.creator)
	s.Require().NoError(err)

	err = f.Stop(s.ctx)
	s.NoError(err)
}

func (s *ForwarderUnitTestSuite) TestDoubleStop() {
	s.watcher.EXPECT().SetHandler(gomock.Any()).Return(nil)
	s.watcher.EXPECT().Start(gomock.Any()).Return(nil)

	f, err := New(s.publisher, s.watcher, s.creator)
	s.Require().NoError(err)

	err = f.Start(s.ctx)
	s.NoError(err)

	s.watcher.EXPECT().Stop(gomock.Any())
	err = f.Stop(s.ctx)
	s.NoError(err)

	err = f.Stop(s.ctx)
	s.NoError(err)
}

func TestForwarderUnitTestSuite(t *testing.T) {
	suite.Run(t, new(ForwarderUnitTestSuite))
}
