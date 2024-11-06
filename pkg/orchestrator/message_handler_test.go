//go:build unit || !integration

package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

type MessageHandlerTestSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	mockStore *jobstore.MockStore
	mockTx    *jobstore.MockTxContext
	handler   *MessageHandler
}

func (suite *MessageHandlerTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockStore = jobstore.NewMockStore(suite.ctrl)
	suite.mockTx = jobstore.NewMockTxContext(suite.ctrl)
	suite.handler = NewMessageHandler(suite.mockStore)
}

func (suite *MessageHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *MessageHandlerTestSuite) TestShouldProcess() {
	suite.True(suite.handler.ShouldProcess(context.Background(), envelope.NewMessage(nil).WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType)))
	suite.True(suite.handler.ShouldProcess(context.Background(), envelope.NewMessage(nil).WithMetadataValue(envelope.KeyMessageType, messages.RunResultMessageType)))
	suite.True(suite.handler.ShouldProcess(context.Background(), envelope.NewMessage(nil).WithMetadataValue(envelope.KeyMessageType, messages.ComputeErrorMessageType)))
	suite.False(suite.handler.ShouldProcess(context.Background(), envelope.NewMessage(nil).WithMetadataValue(envelope.KeyMessageType, "UnknownType")))
}

func (suite *MessageHandlerTestSuite) TestOnBidComplete_Accepted() {
	ctx := context.Background()
	bidResult := &messages.BidResult{
		BaseResponse: messages.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
		Accepted: true,
	}
	message := envelope.NewMessage(bidResult).WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType)

	suite.mockStore.EXPECT().BeginTx(gomock.Any()).Return(suite.mockTx, nil)
	suite.mockStore.EXPECT().UpdateExecution(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockStore.EXPECT().CreateEvaluation(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockTx.EXPECT().Commit().Return(nil)
	suite.mockTx.EXPECT().Rollback().Return(nil)

	err := suite.handler.OnBidComplete(ctx, message)
	suite.NoError(err)
}

func (suite *MessageHandlerTestSuite) TestOnBidComplete_Rejected() {
	ctx := context.Background()
	bidResult := &messages.BidResult{
		BaseResponse: messages.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
		Accepted: false,
	}
	message := envelope.NewMessage(bidResult).WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType)

	suite.mockStore.EXPECT().BeginTx(gomock.Any()).Return(suite.mockTx, nil)
	suite.mockStore.EXPECT().UpdateExecution(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockStore.EXPECT().CreateEvaluation(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockTx.EXPECT().Commit().Return(nil)
	suite.mockTx.EXPECT().Rollback().Return(nil)

	err := suite.handler.OnBidComplete(ctx, message)
	suite.NoError(err)
}

func (suite *MessageHandlerTestSuite) TestOnRunComplete() {
	ctx := context.Background()
	runResult := &messages.RunResult{
		BaseResponse: messages.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
		PublishResult:    &models.SpecConfig{Type: "ipfs"},
		RunCommandResult: &models.RunCommandResult{ExitCode: 0},
	}
	message := envelope.NewMessage(runResult).WithMetadataValue(envelope.KeyMessageType, messages.RunResultMessageType)

	suite.mockStore.EXPECT().BeginTx(gomock.Any()).Return(suite.mockTx, nil)
	suite.mockStore.EXPECT().GetJob(suite.mockTx, "job-1").Return(models.Job{Type: "batch"}, nil)
	suite.mockStore.EXPECT().UpdateExecution(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockStore.EXPECT().CreateEvaluation(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockTx.EXPECT().Commit().Return(nil)
	suite.mockTx.EXPECT().Rollback().Return(nil)

	err := suite.handler.OnRunComplete(ctx, message)
	suite.NoError(err)
}

func (suite *MessageHandlerTestSuite) TestOnComputeFailure() {
	ctx := context.Background()
	computeError := &messages.ComputeError{
		BaseResponse: messages.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
	}
	message := envelope.NewMessage(computeError).WithMetadataValue(envelope.KeyMessageType, messages.ComputeErrorMessageType)

	suite.mockStore.EXPECT().BeginTx(gomock.Any()).Return(suite.mockTx, nil)
	suite.mockStore.EXPECT().UpdateExecution(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockStore.EXPECT().CreateEvaluation(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockTx.EXPECT().Commit().Return(nil)
	suite.mockTx.EXPECT().Rollback().Return(nil)

	err := suite.handler.OnComputeFailure(ctx, message)
	suite.NoError(err)
}

func (suite *MessageHandlerTestSuite) TestOnBidComplete_PropagatesErrors() {
	ctx := context.Background()
	bidResult := &messages.BidResult{
		BaseResponse: messages.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
		Accepted: true,
	}
	message := envelope.NewMessage(bidResult).WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType)

	expectedErr := errors.New("store error")
	suite.mockStore.EXPECT().BeginTx(gomock.Any()).Return(suite.mockTx, nil)
	suite.mockStore.EXPECT().UpdateExecution(suite.mockTx, gomock.Any()).Return(expectedErr)
	suite.mockTx.EXPECT().Rollback().Return(nil)

	err := suite.handler.OnBidComplete(ctx, message)
	suite.ErrorIs(err, expectedErr)
}

func TestMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(MessageHandlerTestSuite))
}
