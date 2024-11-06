//go:build unit || !integration

package watchers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

type BProtocolDispatcherSuite struct {
	suite.Suite
	ctx            context.Context
	ctrl           *gomock.Controller
	computeService *compute.MockEndpoint
	jobStore       *jobstore.MockStore
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
	s.jobStore = jobstore.NewMockStore(s.ctrl)
	s.nodeID = "test-node"
	s.computeErr = errors.New("compute error")
	s.dispatcher = NewBProtocolDispatcher(BProtocolDispatcherParams{
		ID:             s.nodeID,
		ComputeService: s.computeService,
		JobStore:       s.jobStore,
	})
}

func (s *BProtocolDispatcherSuite) TestHandleEvent_InvalidObject() {
	err := s.dispatcher.HandleEvent(s.ctx, watcher.Event{
		Object: "not an execution upsert",
	})
	s.Error(err)
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

	s.computeService.EXPECT().CancelExecution(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, req legacy.CancelExecutionRequest) (legacy.CancelExecutionResponse, error) {
			s.Equal(upsert.Current.ID, req.ExecutionID)
			s.Equal(s.nodeID, req.RoutingMetadata.SourcePeerID)
			s.Equal(upsert.Current.NodeID, req.RoutingMetadata.TargetPeerID)
			return legacy.CancelExecutionResponse{}, nil
		})

	// Expect jobstore update when cancelling
	s.jobStore.EXPECT().UpdateExecution(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, req jobstore.UpdateExecutionRequest) error {
			s.Equal(upsert.Current.ID, req.ExecutionID)
			s.Equal(models.ExecutionStateCancelled, req.NewValues.ComputeState.StateType)
			return nil
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
				s.jobStore.EXPECT().UpdateExecution(s.ctx, gomock.Any()).Return(nil)
			},
			expectErr: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
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
