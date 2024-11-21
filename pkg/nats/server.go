package nats

import (
	"context"
	"os"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
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
		return nil, bacerrors.New("orchestrator port %d is already in use", opts.Port).
			WithComponent(transportServerComponent).
			WithCode(bacerrors.ConfigurationError).
			WithHint("To resolve this, either:\n"+
				"1. Check if you are already running the orchestrator\n"+
				"2. Stop the other process using this port\n"+
				"3. Configure a different port using one of these methods:\n"+
				"   a. Use the `-c %s=<new_port>` flag with your serve command\n"+
				"   b. Set the port in a configuration file with `%s config set %s=<new_port>`",
				types.OrchestratorPortKey, os.Args[0], types.OrchestratorPortKey)
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, bacerrors.Wrap(err, "orchestrator failed to create NATS server").
			WithComponent(transportServerComponent).
			WithCode(bacerrors.ConfigurationError)
	}
	ns.SetLoggerV2(NewZeroLogger(log.Logger, opts.ServerName), opts.Debug, opts.Trace, opts.TraceVerbose)
	go ns.Start()

	if params.ConnectionTimeout == 0 {
		params.ConnectionTimeout = ReadyForConnectionsTimeout
	}
	if !ns.ReadyForConnections(params.ConnectionTimeout) {
		return nil, bacerrors.New("orchestrator NATS not ready for connection within %s", params.ConnectionTimeout).
			WithComponent(transportServerComponent).
			WithCode(bacerrors.ConfigurationError)
	}
	log.Ctx(ctx).Debug().Msgf("NATS server %s listening on %s", ns.ID(), ns.ClientURL())
	return &ServerManager{
		Server: ns,
	}, nil
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
	subsz, err := sm.Server.Subsz(&server.SubszOptions{
		Subscriptions: true,
	})
	if err != nil {
		return models.DebugInfo{}, err
	}
	jsz, err := sm.Server.Jsz(&server.JSzOptions{
		Streams:  true,
		Consumer: true,
	})
	if err != nil {
		return models.DebugInfo{}, err
	}

	return models.DebugInfo{
		Component: "NATSServer",
		Info: map[string]interface{}{
			"ID":         sm.Server.ID(),
			"Varz":       varz,
			"Connz":      connz,
			"Routez":     routez,
			"Subsz":      subsz,
			"JetStreamz": jsz,
		},
	}, nil
}
