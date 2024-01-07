package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/rs/zerolog/log"
)

const ReadyForConnectionsTimeout = 5 * time.Second

type ServerManagerParams struct {
	Options *server.Options
}

// ServerManager is a helper struct to manage a NATS server
type ServerManager struct {
	Server *server.Server
}

// NewServerManager is a helper function to create a NATS server with a given options
func NewServerManager(ctx context.Context, opts *server.Options) (*ServerManager, error) {
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}
	ns.SetLoggerV2(NewZeroLogger(log.Logger, opts.ServerName), opts.Debug, opts.Trace, opts.TraceVerbose)
	go ns.Start()
	if !ns.ReadyForConnections(ReadyForConnectionsTimeout) {
		return nil, fmt.Errorf("could not start nats server on time")
	}
	log.Info().Msgf("NATS server %s listening on %s", ns.ID(), ns.ClientURL())
	return &ServerManager{
		Server: ns,
	}, err
}

// Stop stops the NATS server
func (sm *ServerManager) Stop() {
	sm.Server.Shutdown()
}

// GetDebugInfo returns the debug info of the NATS server
func (sm *ServerManager) GetDebugInfo(ctx context.Context) (model.DebugInfo, error) {
	varz, err := sm.Server.Varz(&server.VarzOptions{})
	if err != nil {
		return model.DebugInfo{}, err
	}
	connz, err := sm.Server.Connz(&server.ConnzOptions{})
	if err != nil {
		return model.DebugInfo{}, err
	}
	routez, err := sm.Server.Routez(&server.RoutezOptions{})
	if err != nil {
		return model.DebugInfo{}, err
	}
	subsz, err := sm.Server.Subsz(&server.SubszOptions{})
	if err != nil {
		return model.DebugInfo{}, err
	}
	return model.DebugInfo{
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
