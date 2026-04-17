//go:build unit || !integration

package watchers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type NCLMessageCreatorTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	protocolRouter *ProtocolRouter
	nodeStore      *nodes.MockLookup
	creator        *NCLMessageCreator
	subjectFn      func(nodeID string) string
}

func TestNCLMessageCreatorTestSuite(t *testing.T) {
	suite.Run(t, new(NCLMessageCreatorTestSuite))
}

func (s *NCLMessageCreatorTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.nodeStore = nodes.NewMockLookup(s.ctrl)
	var err error
	s.protocolRouter, err = NewProtocolRouter(ProtocolRouterParams{
		NodeStore:          s.nodeStore,
		SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2, models.ProtocolNCLV1},
	})
	s.Require().NoError(err)

	s.subjectFn = func(nodeID string) string {
		return "test." + nodeID
	}

	s.creator, err = NewNCLMessageCreator(NCLMessageCreatorParams{
		NodeID:         "test-node",
		ProtocolRouter: s.protocolRouter,
		SubjectFn:      s.subjectFn,
	})
	s.Require().NoError(err)
}

func (s *NCLMessageCreatorTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *NCLMessageCreatorTestSuite) TestNewNCLMessageCreator() {
	tests := []struct {
		name        string
		params      NCLMessageCreatorParams
		expectError string
	}{
		{
			name: "valid params",
			params: NCLMessageCreatorParams{
				NodeID:         "test-node",
				ProtocolRouter: s.protocolRouter,
				SubjectFn:      s.subjectFn,
			},
		},
		{
			name: "missing nodeID",
			params: NCLMessageCreatorParams{
				ProtocolRouter: s.protocolRouter,
				SubjectFn:      s.subjectFn,
			},
			expectError: "nodeID cannot be blank",
		},
		{
			name: "missing protocol router",
			params: NCLMessageCreatorParams{
				NodeID:    "test-node",
				SubjectFn: s.subjectFn,
			},
			expectError: "protocol router cannot be nil",
		},
		{
			name: "missing subject function",
			params: NCLMessageCreatorParams{
				NodeID:         "test-node",
				ProtocolRouter: s.protocolRouter,
			},
			expectError: "subject function cannot be nil",
		},
		{
			name: "blank subject function",
			params: NCLMessageCreatorParams{
				NodeID:         "test-node",
				ProtocolRouter: s.protocolRouter,
				SubjectFn:      func(nodeID string) string { return "" },
			},
			expectError: "subject function returned empty",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			creator, err := NewNCLMessageCreator(tc.params)
			if tc.expectError != "" {
				s.Error(err)
				s.ErrorContains(err, tc.expectError)
				s.Nil(creator)
			} else {
				s.NoError(err)
				s.NotNil(creator)
			}
		})
	}
}

func (s *NCLMessageCreatorTestSuite) TestMessageCreatorFactory() {
	factory := NewNCLMessageCreatorFactory(NCLMessageCreatorFactoryParams{
		ProtocolRouter: s.protocolRouter,
		SubjectFn:      s.subjectFn,
	})

	creator, err := factory.CreateMessageCreator(context.Background(), "test-node")
	s.Require().NoError(err)
	s.NotNil(creator)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_InvalidObject() {
	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: "not an execution upsert",
	})
	s.Error(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_NoStateChange() {
	execution := mock.Execution()
	execution.NodeID = "test-node"
	upsert := models.ExecutionUpsert{
		Previous: execution,
		Current:  execution,
	}

	msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
	s.NoError(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_WrongNode() {
	upsert := setupNewExecution(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateNew,
	)
	upsert.Current.NodeID = "different-node"

	msg, err := s.creator.CreateMessage(createExecutionEvent(upsert))
	s.NoError(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_UnsupportedProtocol() {
	upsert := setupNewExecution(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateNew,
	)
	upsert.Current.NodeID = "test-node"

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
			upsert.Current.NodeID = "test-node"

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
	upsert.Current.NodeID = "test-node"

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
	upsert.Current.NodeID = "test-node"

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
	upsert.Current.NodeID = "test-node"

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

func (s *NCLMessageCreatorTestSuite) TestTransformNetworkConfig() {
	// A helper function to create a test execution with a network configuration
	createExecution := func(networkType models.Network) *models.Execution {
		execution := mock.Execution()
		execution.NodeID = "test-node"
		execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolNCLV1.String()

		// Make sure we have a task with network config
		task := execution.Job.Task()
		if task == nil {
			task = &models.Task{
				Name: "test-task",
			}
			execution.Job.Tasks = []*models.Task{task}
		}

		// Set network configuration
		task.Network = &models.NetworkConfig{
			Type: networkType,
		}

		return execution
	}

	s.Run("host network type transformation", func() {
		// Create an execution with a task that has NetworkHost
		execution := createExecution(models.NetworkHost)

		// Transform network config
		transformed := s.creator.transformNetworkConfig(execution)

		// Verify that NetworkHost was changed to NetworkFull
		s.NotNil(transformed)
		s.NotNil(transformed.Job)
		task := transformed.Job.Task()
		s.NotNil(task)
		s.NotNil(task.Network)
		s.Equal(models.NetworkFull, task.Network.Type)
	})

	s.Run("default network type transformation", func() {
		// Create an execution with a task that has NetworkDefault
		execution := createExecution(models.NetworkDefault)

		// Transform network config
		transformed := s.creator.transformNetworkConfig(execution)

		// Verify that NetworkDefault was changed to nil
		s.NotNil(transformed)
		s.NotNil(transformed.Job)
		task := transformed.Job.Task()
		s.NotNil(task)
		s.Nil(task.Network)
	})

	s.Run("other network types unchanged", func() {
		// Test that other network types are not modified
		testCases := []models.Network{
			models.NetworkBridge,
			models.NetworkNone,
			models.NetworkHTTP,
		}

		for _, networkType := range testCases {
			execution := createExecution(networkType)
			transformed := s.creator.transformNetworkConfig(execution)

			s.NotNil(transformed)
			s.NotNil(transformed.Job)
			task := transformed.Job.Task()
			s.NotNil(task)
			s.NotNil(task.Network)
			s.Equal(networkType, task.Network.Type)
		}
	})

	s.Run("nil network config unchanged", func() {
		// Create an execution with a task that has nil network
		execution := createExecution(models.NetworkBridge)
		execution.Job.Task().Network = nil

		// Transform network config
		transformed := s.creator.transformNetworkConfig(execution)

		// Verify that nil network remains nil
		s.NotNil(transformed)
		s.NotNil(transformed.Job)
		task := transformed.Job.Task()
		s.NotNil(task)
		s.Nil(task.Network)
	})

	s.Run("nil job unchanged", func() {
		// Create an execution with nil Job
		execution := mock.Execution()
		execution.Job = nil

		// Transform network config
		transformed := s.creator.transformNetworkConfig(execution)

		// Verify that nil Job remains nil
		s.NotNil(transformed)
		s.Nil(transformed.Job)
	})

	s.Run("transformation in createAskForBidMessage", func() {
		// Create an execution with NetworkHost that should be transformed
		execution := createExecution(models.NetworkHost)

		// Setup a valid ExecutionUpsert
		upsert := models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		}

		// Call createAskForBidMessage, which should transform the network config
		msg := s.creator.createAskForBidMessage(upsert)

		// Verify the message was created successfully
		s.NotNil(msg)

		// Verify the message type
		s.Equal(messages.AskForBidMessageType, msg.Metadata.Get(envelope.KeyMessageType))

		// Extract the payload and verify it's an AskForBidRequest
		payload, ok := msg.GetPayload(messages.AskForBidRequest{})
		s.True(ok)

		// Verify that the execution in the payload has the transformed network config
		askForBidRequest := payload.(messages.AskForBidRequest)
		s.NotNil(askForBidRequest.Execution)
		s.NotNil(askForBidRequest.Execution.Job)
		task := askForBidRequest.Execution.Job.Task()
		s.NotNil(task)
		s.NotNil(task.Network)

		// The network type should be transformed from Host to Full
		s.Equal(models.NetworkFull, task.Network.Type)
	})
}
