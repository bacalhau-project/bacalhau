package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

const ReadyForConnectionsTimeout = 5 * time.Second

// NewServer is a helper function to create a NATS server with a given options
func NewServer(ctx context.Context, opts *server.Options) (*server.Server, error) {
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}
	go ns.Start()
	if !ns.ReadyForConnections(ReadyForConnectionsTimeout) {
		return nil, fmt.Errorf("nats server not ready")
	}
	log.Info().Msgf("NATS server %s listening on %s", ns.ID(), ns.ClientURL())
	return ns, err
}

// NewClient is a helper function to create a NATS client connection with a given name and servers string
func NewClient(ctx context.Context, name string, servers string) (*nats.Conn, error) {
	nc, err := nats.Connect(servers, nats.Name(name))
	return nc, err
}
