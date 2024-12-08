//go:build unit || !integration

package ncl

import (
	"context"
	"errors"
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
	messages        []*envelope.Message
	failWithMessage string
	mu              sync.Mutex
}

// WithFailureMessage will cause the handler to fail with the given message
func (h *TestMessageHandler) WithFailureMessage(msg string) {
	h.failWithMessage = msg
}

func (h *TestMessageHandler) ShouldProcess(_ context.Context, _ *envelope.Message) bool {
	return true
}

func (h *TestMessageHandler) HandleMessage(_ context.Context, msg *envelope.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failWithMessage != "" {
		return errors.New(h.failWithMessage)
	}
	h.messages = append(h.messages, msg)
	return nil
}

type TestNotifier struct {
	notifications []*envelope.Message
	mu            sync.Mutex
}

func (n *TestNotifier) OnProcessed(ctx context.Context, message *envelope.Message) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.notifications = append(n.notifications, message)
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
