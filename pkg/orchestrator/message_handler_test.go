//go:build unit || !integration

package orchestrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	suite.True(suite.handler.ShouldProcess(context.Background(), ncl.NewMessage(nil).WithMetadataValue(ncl.KeyMessageType, compute.BidResultMessageType)))
	suite.True(suite.handler.ShouldProcess(context.Background(), ncl.NewMessage(nil).WithMetadataValue(ncl.KeyMessageType, compute.RunResultMessageType)))
	suite.True(suite.handler.ShouldProcess(context.Background(), ncl.NewMessage(nil).WithMetadataValue(ncl.KeyMessageType, compute.ComputeErrorMessageType)))
	suite.False(suite.handler.ShouldProcess(context.Background(), ncl.NewMessage(nil).WithMetadataValue(ncl.KeyMessageType, "UnknownType")))
}

func (suite *MessageHandlerTestSuite) TestOnBidComplete_Accepted() {
	ctx := context.Background()
	bidResult := &compute.BidResult{
		BaseResponse: compute.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
		Accepted: true,
	}
	message := ncl.NewMessage(bidResult).WithMetadataValue(ncl.KeyMessageType, compute.BidResultMessageType)

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
	bidResult := &compute.BidResult{
		BaseResponse: compute.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
		Accepted: false,
	}
	message := ncl.NewMessage(bidResult).WithMetadataValue(ncl.KeyMessageType, compute.BidResultMessageType)

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
	runResult := &compute.RunResult{
		BaseResponse: compute.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
		PublishResult:    &models.SpecConfig{Type: "ipfs"},
		RunCommandResult: &models.RunCommandResult{ExitCode: 0},
	}
	message := ncl.NewMessage(runResult).WithMetadataValue(ncl.KeyMessageType, compute.RunResultMessageType)

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
	computeError := &compute.ComputeError{
		BaseResponse: compute.BaseResponse{
			ExecutionID: "exec-1",
			JobID:       "job-1",
			JobType:     "batch",
		},
	}
	message := ncl.NewMessage(computeError).WithMetadataValue(ncl.KeyMessageType, compute.ComputeErrorMessageType)

	suite.mockStore.EXPECT().BeginTx(gomock.Any()).Return(suite.mockTx, nil)
	suite.mockStore.EXPECT().UpdateExecution(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockStore.EXPECT().CreateEvaluation(suite.mockTx, gomock.Any()).Return(nil)
	suite.mockTx.EXPECT().Commit().Return(nil)
	suite.mockTx.EXPECT().Rollback().Return(nil)

	err := suite.handler.OnComputeFailure(ctx, message)
	suite.NoError(err)
}

func TestMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(MessageHandlerTestSuite))
}
