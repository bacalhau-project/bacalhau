//go:build unit || !integration

package stream

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

type BaseTestSuite struct {
	suite.Suite
	natsServer      *server.Server
	natsClient      *nats.Conn
	inboxSubject    string
	streamingClient *Client
	writer          *Writer
	ch              <-chan *concurrency.AsyncResult[[]byte]
	ctx             context.Context
	cancel          context.CancelFunc
}

func (suite *BaseTestSuite) SetupSuite() {
	// Start a NATS server for testing
	opts := &server.Options{Port: -1} // Automatically select a port
	var err error
	suite.natsServer, err = server.NewServer(opts)
	suite.Require().NoError(err)
	suite.natsServer.Start()

	// Wait for the server to be ready
	time.Sleep(1 * time.Second)

	// Create a NATS client
	suite.natsClient, err = nats.Connect(suite.natsServer.ClientURL())
	suite.Require().NoError(err)

	suite.streamingClient, err = NewClient(ClientParams{Conn: suite.natsClient})
	suite.Require().NoError(err)
}

// SetupTest will run before each test in the suite
func (suite *BaseTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// send a request to the setup subject to get the inbox subject, and to start consuming responses
	s := suite.newTestStream()
	suite.inboxSubject = s.replySubject
	suite.ch = s.ch
	suite.writer = s.writer
}

func (suite *BaseTestSuite) TearDownSuite() {
	suite.natsClient.Close()
	suite.natsServer.Shutdown()
}

// testStream is a struct that will be used to test the streaming request
// it holds the dynamic reply subject and the channel that will receive the response
type testStream struct {
	ch           <-chan *concurrency.AsyncResult[[]byte]
	replySubject string
	writer       *Writer
}

// newTestStream will setup a testStream with a random subject
func (suite *BaseTestSuite) newTestStream() *testStream {
	subject := uuid.NewString()
	s := &testStream{}
	_, err := suite.natsClient.Subscribe(subject, func(m *nats.Msg) {
		s.replySubject = m.Reply
	})
	suite.Require().NoError(err)

	ch, err := suite.streamingClient.OpenStream(suite.ctx, subject, []byte("test data"))
	suite.Require().NoError(err)
	suite.Require().NotNil(ch)
	s.ch = ch

	suite.Eventually(func() bool {
		return s.replySubject != ""
	}, 1*time.Second, 10*time.Millisecond, "h.replySubject should not be empty within 1 second")

	s.writer = NewWriter(suite.streamingClient, s.replySubject)
	return s

}

// readResponse reads a response from the channel and returns it
func (suite *BaseTestSuite) readResponse() *concurrency.AsyncResult[[]byte] {
	select {
	case res := <-suite.ch:
		return res
	case <-time.After(200 * time.Millisecond):
		return nil
	}
}

// readResponse reads a response from the channel and returns it
func (suite *BaseTestSuite) readResponseFromStream(s *testStream) *concurrency.AsyncResult[[]byte] {
	select {
	case res := <-s.ch:
		return res
	case <-time.After(200 * time.Millisecond):
		return nil
	}
}
