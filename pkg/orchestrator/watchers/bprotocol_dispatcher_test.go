//go:build unit || !integration

package watchers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type BProtocolDispatcherSuite struct {
	suite.Suite
	ctx            context.Context
	ctrl           *gomock.Controller
	computeService *compute.MockEndpoint
	nodeStore      *nodes.MockLookup
	protocolRouter *ProtocolRouter
	nodeID         string
	computeErr     error
	dispatcher     *BProtocolDispatcher
}

func TestBProtocolDispatcherSuite(t *testing.T) {
	suite.Run(t, new(BProtocolDispatcherSuite))
}

func (s *BProtocolDispatcherSuite) SetupTest() {
	s.ctx = context.Background()
	s.ctrl = gomock.NewController(s.T())
	s.computeService = compute.NewMockEndpoint(s.ctrl)
	s.nodeStore = nodes.NewMockLookup(s.ctrl)
	s.nodeID = "test-node"
	s.computeErr = errors.New("compute error")

	var err error
	s.protocolRouter, err = NewProtocolRouter(ProtocolRouterParams{
		NodeStore:          s.nodeStore,
		SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2, models.ProtocolNCLV1},
	})
	s.Require().NoError(err)

	s.dispatcher = NewBProtocolDispatcher(BProtocolDispatcherParams{
		ID:             s.nodeID,
		ComputeService: s.computeService,
		ProtocolRouter: s.protocolRouter,
	})
}

// Helper to expect protocol router check
func (s *BProtocolDispatcherSuite) expectProtocolSupport(execution *models.Execution) {
	s.nodeStore.EXPECT().Get(s.ctx, execution.NodeID).Return(
		models.NodeState{
			Info: models.NodeInfo{
				SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2},
			},
		}, nil)
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_InvalidObject() {
	err := s.dispatcher.HandleEvent(s.ctx, watcher.Event{
		Object: "not an execution upsert",
	})
	s.Error(err)
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_NoStateChange() {
	// Create upsert with identical Previous and Current
	execution := mock.Execution()
	upsert := models.ExecutionUpsert{
		Previous: execution,
		Current:  execution,
	}

	// No protocol check should happen
	// No compute service calls should happen
	err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_UnsupportedProtocol() {
	// Create new execution that should normally trigger AskForBid
	upsert := setupNewExecution(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateNew,
	)

	// Setup protocol support check to return NCL only
	s.nodeStore.EXPECT().Get(s.ctx, upsert.Current.NodeID).Return(
		models.NodeState{
			Info: models.NodeInfo{
				// Only supports NCL protocol
				SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
			},
		}, nil)

	// No compute service calls should happen
	err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_NoSupportedProtocols() {
	// Create new execution that should normally trigger AskForBid
	upsert := setupNewExecution(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateNew,
	)

	// Setup protocol support check to return empty protocols
	s.nodeStore.EXPECT().Get(s.ctx, upsert.Current.NodeID).Return(
		models.NodeState{
			Info: models.NodeInfo{
				SupportedProtocols: []models.Protocol{}, // No protocols supported
			},
		}, nil)

	// AskForBid should be called
	s.computeService.EXPECT().AskForBid(s.ctx, gomock.Any()).Return(legacy.AskForBidResponse{}, nil)
	err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)

}

func (s *BProtocolDispatcherSuite) TestHandleEvent_AskForBid() {
	tests := []struct {
		name            string
		desiredState    models.ExecutionDesiredStateType
		waitForApproval bool
	}{
		{
			name:            "pending_state",
			desiredState:    models.ExecutionDesiredStatePending,
			waitForApproval: true,
		},
		{
			name:            "running_state",
			desiredState:    models.ExecutionDesiredStateRunning,
			waitForApproval: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			upsert := setupNewExecution(
				tc.desiredState,
				models.ExecutionStateNew,
			)

			s.expectProtocolSupport(upsert.Current)

			s.computeService.EXPECT().AskForBid(s.ctx, gomock.Any()).DoAndReturn(
				func(_ context.Context, req legacy.AskForBidRequest) (legacy.AskForBidResponse, error) {
					s.Equal(upsert.Current, req.Execution)
					s.Equal(tc.waitForApproval, req.WaitForApproval)
					s.Equal(s.nodeID, req.RoutingMetadata.SourcePeerID)
					s.Equal(upsert.Current.NodeID, req.RoutingMetadata.TargetPeerID)
					return legacy.AskForBidResponse{}, nil
				})

			err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(upsert))
			s.NoError(err)
		})
	}
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_BidAccepted() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateAskForBidAccepted,
	)

	s.expectProtocolSupport(upsert.Current)

	s.computeService.EXPECT().BidAccepted(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, req legacy.BidAcceptedRequest) (legacy.BidAcceptedResponse, error) {
			s.Equal(upsert.Current.ID, req.ExecutionID)
			s.Equal(s.nodeID, req.RoutingMetadata.SourcePeerID)
			s.Equal(upsert.Current.NodeID, req.RoutingMetadata.TargetPeerID)
			return legacy.BidAcceptedResponse{}, nil
		})

	err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_BidRejected() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateAskForBidAccepted,
	)

	s.expectProtocolSupport(upsert.Current)

	s.computeService.EXPECT().BidRejected(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, req legacy.BidRejectedRequest) (legacy.BidRejectedResponse, error) {
			s.Equal(upsert.Current.ID, req.ExecutionID)
			s.Equal(s.nodeID, req.RoutingMetadata.SourcePeerID)
			s.Equal(upsert.Current.NodeID, req.RoutingMetadata.TargetPeerID)
			return legacy.BidRejectedResponse{}, nil
		})

	err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_Cancel() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateRunning,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateRunning,
	)

	s.expectProtocolSupport(upsert.Current)

	s.computeService.EXPECT().CancelExecution(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, req legacy.CancelExecutionRequest) (legacy.CancelExecutionResponse, error) {
			s.Equal(upsert.Current.ID, req.ExecutionID)
			s.Equal(s.nodeID, req.RoutingMetadata.SourcePeerID)
			s.Equal(upsert.Current.NodeID, req.RoutingMetadata.TargetPeerID)
			return legacy.CancelExecutionResponse{}, nil
		})

	err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_ComputeErrors() {
	tests := []struct {
		name      string
		upsert    models.ExecutionUpsert
		setupMock func()
		expectErr bool
	}{
		{
			name: "askforbid_error",
			upsert: setupNewExecution(
				models.ExecutionDesiredStatePending,
				models.ExecutionStateNew,
			),
			setupMock: func() {
				s.computeService.EXPECT().
					AskForBid(s.ctx, gomock.Any()).
					Return(legacy.AskForBidResponse{}, s.computeErr)
			},
			expectErr: true,
		},
		{
			name: "bidaccepted_error",
			upsert: setupStateTransition(
				models.ExecutionDesiredStatePending,
				models.ExecutionStateAskForBidAccepted,
				models.ExecutionDesiredStateRunning,
				models.ExecutionStateAskForBidAccepted,
			),
			setupMock: func() {
				s.computeService.EXPECT().
					BidAccepted(s.ctx, gomock.Any()).
					Return(legacy.BidAcceptedResponse{}, s.computeErr)
			},
			expectErr: true,
		},
		{
			name: "bidrejected_error",
			upsert: setupStateTransition(
				models.ExecutionDesiredStatePending,
				models.ExecutionStateAskForBidAccepted,
				models.ExecutionDesiredStateStopped,
				models.ExecutionStateAskForBidAccepted,
			),
			setupMock: func() {
				s.computeService.EXPECT().
					BidRejected(s.ctx, gomock.Any()).
					Return(legacy.BidRejectedResponse{}, s.computeErr)
			},
			expectErr: true,
		},
		{
			name: "cancel_error",
			upsert: setupStateTransition(
				models.ExecutionDesiredStateRunning,
				models.ExecutionStateRunning,
				models.ExecutionDesiredStateStopped,
				models.ExecutionStateRunning,
			),
			setupMock: func() {
				s.computeService.EXPECT().
					CancelExecution(s.ctx, gomock.Any()).
					Return(legacy.CancelExecutionResponse{}, s.computeErr)
			},
			expectErr: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.expectProtocolSupport(tc.upsert.Current)
			tc.setupMock()
			err := s.dispatcher.HandleEvent(s.ctx, createExecutionEvent(tc.upsert))
			if tc.expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}
