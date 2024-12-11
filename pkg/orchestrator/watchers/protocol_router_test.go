//go:build unit || !integration

package watchers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type ProtocolRouterTestSuite struct {
	suite.Suite
	ctx       context.Context
	ctrl      *gomock.Controller
	nodeStore *nodes.MockLookup
	router    *ProtocolRouter
}

func TestProtocolRouterSuite(t *testing.T) {
	suite.Run(t, new(ProtocolRouterTestSuite))
}

func (s *ProtocolRouterTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.ctrl = gomock.NewController(s.T())
	s.nodeStore = nodes.NewMockLookup(s.ctrl)

	// Create router with both protocols supported
	var err error
	s.router, err = NewProtocolRouter(ProtocolRouterParams{
		NodeStore: s.nodeStore,
		SupportedProtocols: []models.Protocol{
			models.ProtocolNCLV1,
			models.ProtocolBProtocolV2,
		},
	})
	s.Require().NoError(err)
}

func (s *ProtocolRouterTestSuite) TestNewProtocolRouter_ValidationErrors() {
	tests := []struct {
		name        string
		params      ProtocolRouterParams
		shouldError bool
	}{
		{
			name: "nil_nodestore",
			params: ProtocolRouterParams{
				SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
			},
			shouldError: true,
		},
		{
			name: "empty_protocols",
			params: ProtocolRouterParams{
				NodeStore:          s.nodeStore,
				SupportedProtocols: []models.Protocol{},
			},
			shouldError: true,
		},
		{
			name: "valid_params",
			params: ProtocolRouterParams{
				NodeStore: s.nodeStore,
				SupportedProtocols: []models.Protocol{
					models.ProtocolNCLV1,
				},
			},
			shouldError: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := NewProtocolRouter(tc.params)
			if tc.shouldError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ProtocolRouterTestSuite) TestPreferredProtocol_NodeStoreError() {
	execution := mock.Execution()

	s.nodeStore.EXPECT().
		Get(s.ctx, execution.NodeID).
		Return(models.NodeState{}, errors.New("node store error"))

	protocol, err := s.router.PreferredProtocol(s.ctx, execution)
	s.Error(err)
	s.Empty(protocol)
}

func (s *ProtocolRouterTestSuite) TestPreferredProtocol_PreferNCL() {
	execution := mock.Execution()

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

	protocol, err := s.router.PreferredProtocol(s.ctx, execution)
	s.NoError(err)
	s.Equal(models.ProtocolNCLV1, protocol)
}

func (s *ProtocolRouterTestSuite) TestPreferredProtocol_FallbackToBProtocol() {
	execution := mock.Execution()

	// Node doesn't advertise any protocols
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{},
		},
	}

	s.nodeStore.EXPECT().Get(s.ctx, execution.NodeID).Return(nodeState, nil)

	protocol, err := s.router.PreferredProtocol(s.ctx, execution)
	s.NoError(err)
	s.Equal(models.ProtocolBProtocolV2, protocol)
}

func (s *ProtocolRouterTestSuite) TestPreferredProtocol_OnlyBProtocol() {
	execution := mock.Execution()

	// Node only supports BProtocol
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2},
		},
	}

	s.nodeStore.EXPECT().
		Get(s.ctx, execution.NodeID).
		Return(nodeState, nil)

	protocol, err := s.router.PreferredProtocol(s.ctx, execution)
	s.NoError(err)
	s.Equal(models.ProtocolBProtocolV2, protocol)
}

func (s *ProtocolRouterTestSuite) TestPreferredProtocol_OnlyNCL() {
	execution := mock.Execution()

	// Node only supports NCL
	nodeState := models.NodeState{
		Info: models.NodeInfo{
			SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
		},
	}

	s.nodeStore.EXPECT().
		Get(s.ctx, execution.NodeID).
		Return(nodeState, nil)

	protocol, err := s.router.PreferredProtocol(s.ctx, execution)
	s.NoError(err)
	s.Equal(models.ProtocolNCLV1, protocol)
}
