package eventhandler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/eventhandler/mock_eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestChainedHandlers(t *testing.T) {
	suite.Run(t, new(jobEventHandlerSuite))
	suite.Run(t, new(localEventHandlerSuite))
}

type jobEventHandlerSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	chainedHandler  *ChainedJobEventHandler
	handler1        *mock_eventhandler.MockJobEventHandler
	handler2        *mock_eventhandler.MockJobEventHandler
	contextProvider *mock_eventhandler.MockContextProvider
	context         context.Context
	event           model.JobEvent
}

// Before each test
func (suite *jobEventHandlerSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.handler1 = mock_eventhandler.NewMockJobEventHandler(suite.ctrl)
	suite.handler2 = mock_eventhandler.NewMockJobEventHandler(suite.ctrl)
	suite.contextProvider = mock_eventhandler.NewMockContextProvider(suite.ctrl)
	suite.chainedHandler = NewChainedJobEventHandler(suite.contextProvider)
	suite.context = context.WithValue(context.Background(), "test", "test")
	suite.event = model.JobEvent{JobID: uuid.NewString()}
}

func (suite *jobEventHandlerSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *jobEventHandlerSuite) TestChainedJobEventHandler_HandleJobEvent() {
	suite.chainedHandler.AddHandlers(suite.handler1, suite.handler2)
	ctx := context.Background()

	// assert context provider is called with the correct context and job id
	suite.contextProvider.EXPECT().GetContext(ctx, suite.event.JobID).Return(suite.context)

	// assert both handlers are called with the context provider's context and event
	gomock.InOrder(
		suite.handler1.EXPECT().HandleJobEvent(suite.context, suite.event).Return(nil),
		suite.handler2.EXPECT().HandleJobEvent(suite.context, suite.event).Return(nil),
	)

	// assert no error was returned
	require.NoError(suite.T(), suite.chainedHandler.HandleJobEvent(ctx, suite.event))
}

func (suite *jobEventHandlerSuite) TestChainedJobEventHandler_HandleJobEventLazilyAdded() {
	suite.chainedHandler.AddHandlers(suite.handler1)
	suite.chainedHandler.AddHandlers(suite.handler2)
	ctx := context.Background()

	// assert context provider is called with the correct context and job id
	suite.contextProvider.EXPECT().GetContext(ctx, suite.event.JobID).Return(suite.context)

	// assert both handlers are called with the context provider's context and event
	gomock.InOrder(
		suite.handler1.EXPECT().HandleJobEvent(suite.context, suite.event).Return(nil),
		suite.handler2.EXPECT().HandleJobEvent(suite.context, suite.event).Return(nil),
	)

	// assert no error was returned
	require.NoError(suite.T(), suite.chainedHandler.HandleJobEvent(ctx, suite.event))
}

func (suite *jobEventHandlerSuite) TestChainedJobEventHandler_HandleJobEventError() {
	suite.chainedHandler.AddHandlers(suite.handler1)
	suite.chainedHandler.AddHandlers(suite.handler2)
	ctx := context.Background()
	mockError := fmt.Errorf("i am an error")

	// assert context provider is called with the correct context and job id
	suite.contextProvider.EXPECT().GetContext(ctx, suite.event.JobID).Return(suite.context)

	// mock first handler to return an error, and don't expect the second handler to be called
	suite.handler1.EXPECT().HandleJobEvent(suite.context, suite.event).Return(mockError)

	// assert no error was returned
	require.Equal(suite.T(), mockError, suite.chainedHandler.HandleJobEvent(ctx, suite.event))
}

func (suite *jobEventHandlerSuite) TestChainedJobEventHandler_HandleJobEventEmptyHandlers() {
	require.Error(suite.T(), suite.chainedHandler.HandleJobEvent(context.Background(), suite.event))
}

////////////////////////////
// local event handler tests
////////////////////////////
type localEventHandlerSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	chainedHandler  *ChainedLocalEventHandler
	handler1        *mock_eventhandler.MockLocalEventHandler
	handler2        *mock_eventhandler.MockLocalEventHandler
	contextProvider *mock_eventhandler.MockContextProvider
	context         context.Context
	event           model.JobLocalEvent
}

// Before each test
func (suite *localEventHandlerSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.handler1 = mock_eventhandler.NewMockLocalEventHandler(suite.ctrl)
	suite.handler2 = mock_eventhandler.NewMockLocalEventHandler(suite.ctrl)
	suite.contextProvider = mock_eventhandler.NewMockContextProvider(suite.ctrl)
	suite.chainedHandler = NewChainedLocalEventHandler(suite.contextProvider)
	suite.context = context.WithValue(context.Background(), "test", "test")
	suite.event = model.JobLocalEvent{JobID: uuid.NewString()}
}

func (suite *localEventHandlerSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *localEventHandlerSuite) TestChainedLocalEventHandler_HandleLocalEvent() {
	suite.chainedHandler.AddHandlers(suite.handler1, suite.handler2)
	ctx := context.Background()

	// assert context provider is called with the correct context and local id
	suite.contextProvider.EXPECT().GetContext(ctx, suite.event.JobID).Return(suite.context)

	// assert both handlers are called with the context provider's context and event
	gomock.InOrder(
		suite.handler1.EXPECT().HandleLocalEvent(suite.context, suite.event).Return(nil),
		suite.handler2.EXPECT().HandleLocalEvent(suite.context, suite.event).Return(nil),
	)

	// assert no error was returned
	require.NoError(suite.T(), suite.chainedHandler.HandleLocalEvent(ctx, suite.event))
}

func (suite *localEventHandlerSuite) TestChainedLocalEventHandler_HandleLocalEventLazilyAdded() {
	suite.chainedHandler.AddHandlers(suite.handler1)
	suite.chainedHandler.AddHandlers(suite.handler2)
	ctx := context.Background()

	// assert context provider is called with the correct context and local id
	suite.contextProvider.EXPECT().GetContext(ctx, suite.event.JobID).Return(suite.context)

	// assert both handlers are called with the context provider's context and event
	gomock.InOrder(
		suite.handler1.EXPECT().HandleLocalEvent(suite.context, suite.event).Return(nil),
		suite.handler2.EXPECT().HandleLocalEvent(suite.context, suite.event).Return(nil),
	)

	// assert no error was returned
	require.NoError(suite.T(), suite.chainedHandler.HandleLocalEvent(ctx, suite.event))
}

func (suite *localEventHandlerSuite) TestChainedLocalEventHandler_HandleLocalEventError() {
	suite.chainedHandler.AddHandlers(suite.handler1)
	suite.chainedHandler.AddHandlers(suite.handler2)
	ctx := context.Background()
	mockError := fmt.Errorf("i am an error")

	// assert context provider is called with the correct context and local id
	suite.contextProvider.EXPECT().GetContext(ctx, suite.event.JobID).Return(suite.context)

	// mock first handler to return an error, and don't expect the second handler to be called
	suite.handler1.EXPECT().HandleLocalEvent(suite.context, suite.event).Return(mockError)

	// assert no error was returned
	require.Equal(suite.T(), mockError, suite.chainedHandler.HandleLocalEvent(ctx, suite.event))
}

func (suite *localEventHandlerSuite) TestChainedLocalEventHandler_HandleLocalEventEmptyHandlers() {
	require.Error(suite.T(), suite.chainedHandler.HandleLocalEvent(context.Background(), suite.event))
}
