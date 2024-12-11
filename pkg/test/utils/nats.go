package testutils

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
)

// startNatsOnPort will start a NATS server on a specific port and return a server and client instances
func startNatsOnPort(t *testing.T, port int) *natsserver.Server {
	t.Helper()
	opts := &natstest.DefaultTestOptions
	opts.Port = port

	natsServer := natstest.RunServer(opts)
	return natsServer
}

func StartNatsServer(t *testing.T) *natsserver.Server {
	t.Helper()
	port, err := network.GetFreePort()
	require.NoError(t, err)

	return startNatsOnPort(t, port)
}

func CreateNatsClient(t *testing.T, url string) *nats.Conn {
	nc, err := nats.Connect(url,
		nats.ReconnectBufSize(-1),                 // disable reconnect buffer so client fails fast if disconnected
		nats.ReconnectWait(200*time.Millisecond),  //nolint:mnd // reduce reconnect wait to fail fast in tests
		nats.FlusherTimeout(100*time.Millisecond), //nolint:mnd // reduce flusher timeout to speed up tests
	)
	require.NoError(t, err)
	return nc
}

// StartNats will start a NATS server on a random port and return a server and client instances
func StartNats(t *testing.T) (*natsserver.Server, *nats.Conn) {
	natsServer := StartNatsServer(t)
	return natsServer, CreateNatsClient(t, natsServer.ClientURL())
}

// RestartNatsServer will restart the NATS server and return a new server and client using the same port
func RestartNatsServer(t *testing.T, natsServer *natsserver.Server) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	natsServer.Shutdown()

	u, err := url.Parse(natsServer.ClientURL())
	require.NoError(t, err, "Failed to parse NATS server URL %s", natsServer.ClientURL())

	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err, "Failed to convert port %s to int", u.Port())

	natsServer = startNatsOnPort(t, port)
	return natsServer, CreateNatsClient(t, natsServer.ClientURL())
}
