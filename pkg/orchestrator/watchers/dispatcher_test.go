//go:build unit || !integration

package watchers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type DispatcherTestSuite struct {
	suite.Suite
	ctx                   context.Context
	ctrl                  *gomock.Controller
	nodeStore             *routing.MockNodeInfoStore
	nclProtocolDispatcher *watcher.MockEventHandler
	bProtocolDispatcher   *watcher.MockEventHandler
	dispatcher            *Dispatcher
}

func TestDispatcherSuite(t *testing.T) {
	suite.Run(t, new(DispatcherTestSuite))
}

func (s *DispatcherTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.ctrl = gomock.NewController(s.T())
	s.nodeStore = routing.NewMockNodeInfoStore(s.ctrl)
	s.nclProtocolDispatcher = watcher.NewMockEventHandler(s.ctrl)
	s.bProtocolDispatcher = watcher.NewMockEventHandler(s.ctrl)

	dispatchers := map[models.Protocol]watcher.EventHandler{
		models.ProtocolNCLV1:       s.nclProtocolDispatcher,
		models.ProtocolBProtocolV2: s.bProtocolDispatcher,
	}

	var err error
	s.dispatcher, err = NewDispatcher(DispatcherParams{
		NodeStore:   s.nodeStore,
		Dispatchers: dispatchers,
	})
	s.Require().NoError(err)
}

func (s *DispatcherTestSuite) TestNewDispatcher_ValidationErrors() {
	tests := []struct {
		name   string
		params DispatcherParams
		// Add field to track if the param set should error
		shouldError bool
	}{
		{
			name: "nil_nodestore",
			params: DispatcherParams{
				// Remove NodeStore completely rather than set to nil
				Dispatchers: map[models.Protocol]watcher.EventHandler{"test": s.nclProtocolDispatcher},
			},
			shouldError: true,
		},
		{
			name: "empty_dispatchers",
			params: DispatcherParams{
				NodeStore:   s.nodeStore,
				Dispatchers: map[models.Protocol]watcher.EventHandler{},
			},
			shouldError: true,
		},
		{
			name: "nil_dispatcher",
			params: DispatcherParams{
				NodeStore: s.nodeStore,
				Dispatchers: map[models.Protocol]watcher.EventHandler{
					"test": nil,
				},
			},
			shouldError: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := NewDispatcher(tc.params)
			if tc.shouldError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *DispatcherTestSuite) TestHandleEvent_InvalidObject() {
	err := s.dispatcher.HandleEvent(s.ctx, watcher.Event{
		Object: "not an execution upsert",
	})
	s.Error(err)
}

func (s *DispatcherTestSuite) TestHandleEvent_NodeStoreError() {
	execution := mock.Execution()
	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}

	s.nodeStore.EXPECT().
		Get(s.ctx, execution.NodeID).
		Return(models.NodeState{}, errors.New("node store error"))

	err := s.dispatcher.HandleEvent(s.ctx, event)
	s.Error(err)
}

// TestHandleEvent_PreferBProtocol tests that the dispatcher will prefer the BProtocol by default
func (s *DispatcherTestSuite) TestHandleEvent_PreferBProtocol() {
	execution := mock.Execution()
	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}

	// Node supports both protocols
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{
				models.ProtocolNCLV1,
				models.ProtocolBProtocolV2,
			},
		},
	}

	s.nodeStore.EXPECT().Get(s.ctx, execution.NodeID).Return(nodeState, nil)

	// NCL should be preferred and used
	s.bProtocolDispatcher.EXPECT().HandleEvent(s.ctx, event).Return(nil)

	err := s.dispatcher.HandleEvent(s.ctx, event)
	s.NoError(err)
}

// TestHandleEvent_PreferNCL tests that the dispatcher will prefer NCL if the environment variable is set
func (s *DispatcherTestSuite) TestHandleEvent_PreferNCL() {
	s.T().Setenv(models.EnvPreferNCL, "true")
	execution := mock.Execution()
	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}

	// Node supports both protocols
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{
				models.ProtocolNCLV1,
				models.ProtocolBProtocolV2,
			},
		},
	}

	s.nodeStore.EXPECT().Get(s.ctx, execution.NodeID).Return(nodeState, nil)

	// NCL should be preferred and used
	s.nclProtocolDispatcher.EXPECT().HandleEvent(s.ctx, event).Return(nil)

	err := s.dispatcher.HandleEvent(s.ctx, event)
	s.NoError(err)
}

func (s *DispatcherTestSuite) TestHandleEvent_FallbackToBProtocol() {
	execution := mock.Execution()
	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}

	// Node doesn't advertise any protocols
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{},
		},
	}

	s.nodeStore.EXPECT().Get(s.ctx, execution.NodeID).Return(nodeState, nil)

	// Should fall back to BProtocol
	s.bProtocolDispatcher.EXPECT().HandleEvent(s.ctx, event).Return(nil)

	err := s.dispatcher.HandleEvent(s.ctx, event)
	s.NoError(err)
}

func (s *DispatcherTestSuite) TestHandleEvent_OnlyBProtocol() {
	execution := mock.Execution()
	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}

	// Node only supports BProtocol
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2},
		},
	}

	s.nodeStore.EXPECT().
		Get(s.ctx, execution.NodeID).
		Return(nodeState, nil)

	s.bProtocolDispatcher.EXPECT().
		HandleEvent(s.ctx, event).
		Return(nil)

	err := s.dispatcher.HandleEvent(s.ctx, event)
	s.NoError(err)
}

func (s *DispatcherTestSuite) TestHandleEvent_OnlyNCL() {
	execution := mock.Execution()
	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}

	// Node only supports NCL
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
		},
	}

	s.nodeStore.EXPECT().
		Get(s.ctx, execution.NodeID).
		Return(nodeState, nil)

	s.nclProtocolDispatcher.EXPECT().
		HandleEvent(s.ctx, event).
		Return(nil)

	err := s.dispatcher.HandleEvent(s.ctx, event)
	s.NoError(err)
}

func (s *DispatcherTestSuite) TestHandleEvent_DispatcherError() {
	execution := mock.Execution()
	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}

	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
		},
	}

	s.nodeStore.EXPECT().
		Get(s.ctx, execution.NodeID).
		Return(nodeState, nil)

	s.nclProtocolDispatcher.EXPECT().
		HandleEvent(s.ctx, event).
		Return(errors.New("dispatcher error"))

	err := s.dispatcher.HandleEvent(s.ctx, event)
	s.Error(err)
}
