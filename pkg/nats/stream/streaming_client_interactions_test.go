//go:build integration || !unit

package stream

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	nats_helper "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

const (
	subjectName = "topic.stream"
	testString  = "Hello from bacalhau"
)

type StreamingClientInteractionTestSuite struct {
	suite.Suite

	ctx       context.Context
	natServer *server.Server
	pc        *ProducerClient
	cc        *ConsumerClient
}

type testData struct {
	contextCancelled    bool
	heartBeatRequestSub string
}

func (s *StreamingClientInteractionTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.natServer = s.createNatsServer()
	s.pc = s.createProducerClient()
	s.cc = s.createConsumerClient()
}

func (s *StreamingClientInteractionTestSuite) TearDownSuite() {
	s.cc.Conn.Close()
	s.pc.Conn.Close()
	s.natServer.Shutdown()
}

func (s *StreamingClientInteractionTestSuite) createNatsServer() *server.Server {
	port, err := network.GetFreePort()
	s.Require().NoError(err)

	serverOpts := server.Options{
		Port: port,
	}

	ns, err := nats_helper.NewServerManager(s.ctx, nats_helper.ServerManagerParams{
		Options: &serverOpts,
	})
	s.Require().NoError(err)

	return ns.Server
}

func (s *StreamingClientInteractionTestSuite) createProducerClient() *ProducerClient {
	clientManager, err := nats_helper.NewClientManager(s.ctx, s.natServer.ClientURL(), nats.Name("streaming-test"))
	s.Require().NoError(err)

	pc, err := NewProducerClient(s.ctx, ProducerClientParams{
		Conn: clientManager.Client,
		Config: StreamProducerClientConfig{
			HeartBeatIntervalDuration:        200 * time.Millisecond,
			HeartBeatRequestTimeout:          100 * time.Millisecond,
			StreamCancellationBufferDuration: 200 * time.Millisecond,
		},
	})

	s.Require().NoError(err)
	s.Eventually(func() bool {
		return pc.Conn.IsConnected()
	}, 500*time.Millisecond, 10*time.Millisecond)
	return pc
}

func (s *StreamingClientInteractionTestSuite) createConsumerClient() *ConsumerClient {
	clientManager, err := nats_helper.NewClientManager(s.ctx, s.natServer.ClientURL(), nats.Name("streaming-test"))
	s.Require().NoError(err)

	cc, err := NewConsumerClient(ConsumerClientParams{
		Conn: clientManager.Client,
		Config: StreamConsumerClientConfig{
			StreamCancellationBufferDuration: 100 * time.Millisecond,
		},
	})

	s.Require().NoError(err)
	s.Eventually(func() bool {
		return cc.Conn.IsConnected()
	}, 500*time.Millisecond, 10*time.Millisecond)
	return cc
}

func TestStreamingClientTestSuite(t *testing.T) {
	suite.Run(t, new(StreamingClientInteractionTestSuite))
}

func (s *StreamingClientInteractionTestSuite) TestStreamConsumerClientGoingDown() {
	// Set up for the test
	ctx := context.Background()
	td := &testData{}
	clientManager, err := nats_helper.NewClientManager(ctx, s.natServer.ClientURL(), nats.Name("stream-testing-consumer-going-down"))
	s.Require().NoError(err)

	// Produce some data once asked for
	ctx, cancel := context.WithCancel(ctx)
	_, err = clientManager.Client.Subscribe(subjectName, func(msg *nats.Msg) {
		s.Require().NotNil(msg)

		var streamRequest Request
		err := json.Unmarshal(msg.Data, &streamRequest)
		s.Require().NoError(err)

		err = s.pc.AddStream(
			streamRequest.ConsumerID,
			streamRequest.StreamID,
			msg.Subject,
			streamRequest.HeartBeatRequestSub,
			cancel,
		)
		s.Require().NoError(err)

		td.heartBeatRequestSub = streamRequest.HeartBeatRequestSub
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					s.Require().NotNil(ctx.Err())
					td.contextCancelled = true
					return
				case <-ticker.C:
					data, err := json.Marshal(testString)
					s.Require().NoError(err)

					sMsg := StreamingMsg{
						Type: 1,
						Data: data,
					}

					sMsgData, err := json.Marshal(sMsg)
					s.Require().NoError(err)

					clientManager.Client.Publish(msg.Reply, sMsgData)
				}
			}
		}()
	})
	s.Require().NoError(err)
	data, err := json.Marshal(testString)
	s.Require().NoError(err)

	_, err = s.cc.OpenStream(s.ctx, subjectName, data)
	s.Require().NoError(err)

	s.Eventually(func() bool {
		return td.heartBeatRequestSub != ""
	}, time.Second*5, time.Millisecond*100, "Streaming request yet not received")

	// Close the Consumer Client After Certain Time
	time.Sleep(time.Second * 1)
	s.cc.Conn.Close()

	// Validate that producer client does the cleanup
	s.Eventually(func() bool {
		return td.contextCancelled
	}, 2800*time.Millisecond, 50*time.Millisecond)
}
