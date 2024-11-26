//go:build unit || !integration

package watchers

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type NCLMessageCreatorTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	protocolRouter *ProtocolRouter
	nodeStore      *routing.MockNodeInfoStore
	creator        *NCLMessageCreator
	subjectFn      func(nodeID string) string
}

func TestNCLMessageCreatorTestSuite(t *testing.T) {
	suite.Run(t, new(NCLMessageCreatorTestSuite))
}

func (s *NCLMessageCreatorTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.nodeStore = routing.NewMockNodeInfoStore(s.ctrl)
	var err error
	s.protocolRouter, err = NewProtocolRouter(ProtocolRouterParams{
		NodeStore:          s.nodeStore,
		SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2, models.ProtocolNCLV1},
	})
	s.Require().NoError(err)

	s.subjectFn = func(nodeID string) string {
		return "test." + nodeID
	}

	s.creator = NewNCLMessageCreator(NCLMessageCreatorParams{
		ProtocolRouter: s.protocolRouter,
		SubjectFn:      s.subjectFn,
	})
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_InvalidObject() {
	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: "not an execution upsert",
	})
	s.Error(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_NoStateChange() {
	upsert := models.ExecutionUpsert{
		Previous: mock.Execution(),
		Current:  mock.Execution(),
	}

	msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
	s.NoError(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_UnsupportedProtocol() {
	upsert := setupNewExecution(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateNew,
	)

	// Mock node only supporting BProtocol
	s.nodeStore.EXPECT().Get(gomock.Any(), upsert.Current.NodeID).Return(
		models.NodeState{
			Info: models.NodeInfo{
				SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2},
			},
		}, nil)

	msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
	s.NoError(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_AskForBid() {
	tests := []struct {
		name         string
		desiredState models.ExecutionDesiredStateType
	}{
		{
			name:         "pending_state",
			desiredState: models.ExecutionDesiredStatePending,
		},
		{
			name:         "running_state",
			desiredState: models.ExecutionDesiredStateRunning,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			upsert := setupNewExecution(
				tc.desiredState,
				models.ExecutionStateNew,
			)

			// Mock node supporting NCL
			s.nodeStore.EXPECT().Get(gomock.Any(), upsert.Current.NodeID).Return(
				models.NodeState{
					Info: models.NodeInfo{
						SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
					},
				}, nil)

			msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
			s.Require().NoError(err)
			s.Require().NotNil(msg)

			s.Equal(messages.AskForBidMessageType, msg.Metadata.Get(envelope.KeyMessageType))
			s.Equal(s.subjectFn(upsert.Current.NodeID), msg.Metadata.Get(ncl.KeySubject))

			payload, ok := msg.GetPayload(messages.AskForBidRequest{})
			s.Require().True(ok)
			request := payload.(messages.AskForBidRequest)
			s.Equal(upsert.Current.ID, request.Execution.ID)
			s.Equal(upsert.Events, request.Events)
		})
	}
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_BidAccepted() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateAskForBidAccepted,
	)

	// Mock node supporting NCL
	s.nodeStore.EXPECT().Get(gomock.Any(), upsert.Current.NodeID).Return(
		models.NodeState{
			Info: models.NodeInfo{
				SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
			},
		}, nil)

	msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
	s.Require().NoError(err)
	s.Require().NotNil(msg)

	s.Equal(messages.BidAcceptedMessageType, msg.Metadata.Get(envelope.KeyMessageType))
	s.Equal(s.subjectFn(upsert.Current.NodeID), msg.Metadata.Get(ncl.KeySubject))

	payload, ok := msg.GetPayload(messages.BidAcceptedRequest{})
	s.Require().True(ok)
	request := payload.(messages.BidAcceptedRequest)
	s.Equal(upsert.Current.ID, request.ExecutionID)
	s.Equal(upsert.Events, request.Events)
	s.True(request.Accepted)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_BidRejected() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateAskForBidAccepted,
	)

	// Mock node supporting NCL
	s.nodeStore.EXPECT().Get(gomock.Any(), upsert.Current.NodeID).Return(
		models.NodeState{
			Info: models.NodeInfo{
				SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
			},
		}, nil)

	msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
	s.Require().NoError(err)
	s.Require().NotNil(msg)

	s.Equal(messages.BidRejectedMessageType, msg.Metadata.Get(envelope.KeyMessageType))
	s.Equal(s.subjectFn(upsert.Current.NodeID), msg.Metadata.Get(ncl.KeySubject))

	payload, ok := msg.GetPayload(messages.BidRejectedRequest{})
	s.Require().True(ok)
	request := payload.(messages.BidRejectedRequest)
	s.Equal(upsert.Current.ID, request.ExecutionID)
	s.Equal(upsert.Events, request.Events)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_Cancel() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateRunning,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateRunning,
	)

	// Mock node supporting NCL
	s.nodeStore.EXPECT().Get(gomock.Any(), upsert.Current.NodeID).Return(
		models.NodeState{
			Info: models.NodeInfo{
				SupportedProtocols: []models.Protocol{models.ProtocolNCLV1},
			},
		}, nil)

	msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
	s.Require().NoError(err)
	s.Require().NotNil(msg)

	s.Equal(messages.CancelExecutionMessageType, msg.Metadata.Get(envelope.KeyMessageType))
	s.Equal(s.subjectFn(upsert.Current.NodeID), msg.Metadata.Get(ncl.KeySubject))

	payload, ok := msg.GetPayload(messages.CancelExecutionRequest{})
	s.Require().True(ok)
	request := payload.(messages.CancelExecutionRequest)
	s.Equal(upsert.Current.ID, request.ExecutionID)
	s.Equal(upsert.Events, request.Events)
}
