//go:build unit || !integration

package ncl

import (
	"context"
	"sync"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

const (
	TestPayloadType       = "TestPayload"
	TestSubject           = "a.subject"
	TestDestinationPrefix = "b.prefix"
)

type TestPayload struct {
	Message string
	Value   int
}

type TestMessageHandler struct {
	messages []*envelope.Message
	mu       sync.Mutex
}

func (h *TestMessageHandler) ShouldProcess(_ context.Context, _ *envelope.Message) bool {
	return true
}

func (h *TestMessageHandler) HandleMessage(_ context.Context, msg *envelope.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, msg)
	return nil
}

// StartNats will start a NATS server on a random port and return a server and client instances
func StartNats(t *testing.T) (*server.Server, *nats.Conn) {
	t.Helper()
	opts := &natstest.DefaultTestOptions
	opts.Port = -1

	natsServer := natstest.RunServer(opts)
	nc, err := nats.Connect(natsServer.ClientURL())
	require.NoError(t, err)
	return natsServer, nc
}
