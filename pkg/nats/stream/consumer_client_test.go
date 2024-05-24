//go:build unit || !integration

package stream

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	BaseTestSuite
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.natsClient.Close()
	suite.natsServer.Shutdown()
}

// TestMultipleStreams tests that multiple streams can be read from with no interference
func (suite *ClientTestSuite) TestMultipleStreams() {
	stream1 := suite.newTestStream()
	stream2 := suite.newTestStream()

	data1 := []byte("test data 1")
	data2 := []byte("test data 2")

	_, err := stream1.writer.Write(data1)
	suite.Require().NoError(err)

	_, err = stream2.writer.Write(data2)
	suite.Require().NoError(err)

	// Verify
	response1 := suite.readResponseFromStream(stream1)
	suite.Require().NoError(response1.Err)
	suite.Require().Equal(data1, response1.Value, "Expected response to be equal to the data")

	response2 := suite.readResponseFromStream(stream2)
	suite.Require().NoError(response2.Err)
	suite.Require().Equal(data2, response2.Value, "Expected response to be equal to the data")

	// Verify no more records
	suite.Require().Nil(suite.readResponseFromStream(stream1), "Expected no more responses after the first one")
	suite.Require().Nil(suite.readResponseFromStream(stream2), "Expected no more responses after the first one")
}

func (suite *ClientTestSuite) TestContextCancel() {
	suite.cancel()
	// ctx cancellation will be detected after a new record is streamed
	_, err := suite.writer.Write([]byte("test"))
	suite.Require().NoError(err)

	select {
	case _, ok := <-suite.ch:
		suite.Require().False(ok, "Expected the channel to be closed")
	case <-time.After(1 * time.Second):
		suite.Fail("Timeout waiting for the channel to be closed")
	}
}

func (suite *ClientTestSuite) TestRequestWithContextCancellation() {
	// Test that a request is properly cancelled by context
	subj := "test.cancel"
	ctx, cancel := context.WithCancel(context.Background())
	payload := []byte("cancel test payload")

	// Cancel context before making request
	cancel()

	// Attempt to make the request
	_, err := suite.streamingClient.OpenStream(ctx, subj, "", payload)
	suite.Require().Error(err, "Expected an error due to cancelled context")
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
