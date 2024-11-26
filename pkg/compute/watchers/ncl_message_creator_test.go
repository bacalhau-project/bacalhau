//go:build unit || !integration

package watchers

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type NCLMessageCreatorTestSuite struct {
	suite.Suite
	creator *NCLMessageCreator
}

func TestNCLMessageCreatorTestSuite(t *testing.T) {
	suite.Run(t, new(NCLMessageCreatorTestSuite))
}

func (s *NCLMessageCreatorTestSuite) SetupTest() {
	s.creator = NewNCLMessageCreator()
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_InvalidObject() {
	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: "not an execution upsert",
	})
	s.Error(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_WrongProtocol() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolBProtocolV2.String()

	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	})
	s.NoError(err)
	s.Nil(msg)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_BidAccepted() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolNCLV1.String()
	execution.ComputeState = models.State[models.ExecutionStateType]{
		StateType: models.ExecutionStateAskForBidAccepted,
	}

	event := models.Event{Topic: "test-topic", Message: "test-message"}

	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{&event},
		},
	})

	s.Require().NoError(err)
	s.Require().NotNil(msg)

	s.Equal(messages.BidResultMessageType, msg.Metadata.Get(envelope.KeyMessageType))

	payload, ok := msg.GetPayload(messages.BidResult{})
	s.Require().True(ok)
	result := payload.(messages.BidResult)

	s.True(result.Accepted)
	s.Equal(execution.ID, result.ExecutionID)
	s.Equal(execution.JobID, result.JobID)
	s.Equal(execution.Job.Type, result.JobType)
	s.Equal([]*models.Event{&event}, result.Events)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_BidRejected() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolNCLV1.String()
	execution.ComputeState = models.State[models.ExecutionStateType]{
		StateType: models.ExecutionStateAskForBidRejected,
	}

	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	})

	s.Require().NoError(err)
	s.Require().NotNil(msg)

	s.Equal(messages.BidResultMessageType, msg.Metadata.Get(envelope.KeyMessageType))

	payload, ok := msg.GetPayload(messages.BidResult{})
	s.Require().True(ok)
	result := payload.(messages.BidResult)

	s.False(result.Accepted)
	s.Equal(execution.ID, result.ExecutionID)
	s.Equal(execution.JobID, result.JobID)
	s.Equal(execution.Job.Type, result.JobType)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_ExecutionCompleted() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolNCLV1.String()
	execution.ComputeState = models.State[models.ExecutionStateType]{
		StateType: models.ExecutionStateCompleted,
	}
	execution.PublishedResult = &models.SpecConfig{Type: "myResult"}
	execution.RunOutput = &models.RunCommandResult{ExitCode: 0}

	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	})

	s.Require().NoError(err)
	s.Require().NotNil(msg)

	s.Equal(messages.RunResultMessageType, msg.Metadata.Get(envelope.KeyMessageType))

	payload, ok := msg.GetPayload(messages.RunResult{})
	s.Require().True(ok)
	result := payload.(messages.RunResult)

	s.Equal(execution.ID, result.ExecutionID)
	s.Equal(execution.JobID, result.JobID)
	s.Equal(execution.Job.Type, result.JobType)
	s.Equal("myResult", result.PublishResult.Type)
	s.Equal(0, result.RunCommandResult.ExitCode)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_ExecutionFailed() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolNCLV1.String()
	execution.ComputeState = models.State[models.ExecutionStateType]{
		StateType: models.ExecutionStateFailed,
	}

	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	})

	s.Require().NoError(err)
	s.Require().NotNil(msg)

	s.Equal(messages.ComputeErrorMessageType, msg.Metadata.Get(envelope.KeyMessageType))

	payload, ok := msg.GetPayload(messages.ComputeError{})
	s.Require().True(ok)
	result := payload.(messages.ComputeError)

	s.Equal(execution.ID, result.ExecutionID)
	s.Equal(execution.JobID, result.JobID)
	s.Equal(execution.Job.Type, result.JobType)
}

func (s *NCLMessageCreatorTestSuite) TestCreateMessage_UnhandledState() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = models.ProtocolNCLV1.String()
	execution.ComputeState = models.State[models.ExecutionStateType]{
		StateType: models.ExecutionStateNew,
	}

	msg, err := s.creator.CreateMessage(watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	})

	s.NoError(err)
	s.Nil(msg)
}
