package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/rs/zerolog/log"
)

const ReadyForConnectionsTimeout = 5 * time.Second

type ServerManagerParams struct {
	Options           *server.Options
	ConnectionTimeout time.Duration
}

// ServerManager is a helper struct to manage a NATS server
type ServerManager struct {
	Server *server.Server
}

// NewServerManager is a helper function to create a NATS server with a given options
func NewServerManager(ctx context.Context, params ServerManagerParams) (*ServerManager, error) {
	opts := params.Options

	// If the port we want to use is already running (or the port is in use) then bail
	if !network.IsPortOpen(opts.Port) {
		return nil, fmt.Errorf("port %d is already in use", opts.Port)
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}
	ns.SetLoggerV2(NewZeroLogger(log.Logger, opts.ServerName), opts.Debug, opts.Trace, opts.TraceVerbose)
	go ns.Start()

	if params.ConnectionTimeout == 0 {
		params.ConnectionTimeout = ReadyForConnectionsTimeout
	}
	if !ns.ReadyForConnections(params.ConnectionTimeout) {
		return nil, fmt.Errorf("could not start nats server on time")
	}
	log.Ctx(ctx).Info().Msgf("NATS server %s listening on %s", ns.ID(), ns.ClientURL())
	return &ServerManager{
		Server: ns,
	}, err
}

// Stop stops the NATS server
func (sm *ServerManager) Stop() {
	sm.Server.Shutdown()
}

// GetDebugInfo returns the debug info of the NATS server
func (sm *ServerManager) GetDebugInfo(ctx context.Context) (models.DebugInfo, error) {
	varz, err := sm.Server.Varz(&server.VarzOptions{})
	if err != nil {
		return models.DebugInfo{}, err
	}
	connz, err := sm.Server.Connz(&server.ConnzOptions{})
	if err != nil {
		return models.DebugInfo{}, err
	}
	routez, err := sm.Server.Routez(&server.RoutezOptions{})
	if err != nil {
		return models.DebugInfo{}, err
	}
	subsz, err := sm.Server.Subsz(&server.SubszOptions{})
	if err != nil {
		return models.DebugInfo{}, err
	}
	return models.DebugInfo{
		Component: "NATSServer",
		Info: map[string]interface{}{
			"ID":     sm.Server.ID(),
			"Varz":   varz,
			"Connz":  connz,
			"Routez": routez,
			"Subsz":  subsz,
		},
	}, nil
}
