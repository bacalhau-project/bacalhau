package nats

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/nats-io/nats.go"
)

type ClientManager struct {
	Client *nats.Conn
}

// NewClientManager is a helper function to create a NATS client connection with a given name and servers string
func NewClientManager(ctx context.Context, servers string, options ...nats.Option) (*ClientManager, error) {
	nc, err := nats.Connect(servers, options...)
	if err != nil {
		return nil, handleConnectionError(err, servers)
	}
	return &ClientManager{
		Client: nc,
	}, nil
}

// Stop stops the NATS client
func (cm *ClientManager) Stop() {
	cm.Client.Close()
}

// DebugInfo returns the debug info of the NATS client
func (cm *ClientManager) GetDebugInfo(ctx context.Context) (models.DebugInfo, error) {
	stats := cm.Client.Stats()
	servers := cm.Client.Servers()
	buffered, err := cm.Client.Buffered()
	if err != nil {
		return models.DebugInfo{}, err
	}

	return models.DebugInfo{
		Component: "NATSClient",
		Info: map[string]interface{}{
			"Name":     cm.Client.Opts.Name,
			"Stats":    stats,
			"Servers":  servers,
			"Buffered": buffered,
			"Connection": map[string]interface{}{
				"IsConnected": cm.Client.IsConnected(),
				"Addr":        cm.Client.ConnectedAddr(),
				"Url":         cm.Client.ConnectedUrl(),
				"ServerId":    cm.Client.ConnectedServerId(),
				"ServerName":  cm.Client.ConnectedServerName(),
				"ClusterName": cm.Client.ConnectedClusterName(),
			},
		},
	}, nil
}

func handleConnectionError(err error, servers string) error {
	switch {
	case errors.Is(err, nats.ErrNoServers):
		defaultServers := strings.Join(types.Default.Compute.Orchestrators, ",")
		hint := fmt.Sprintf(`to resolve this, either:
1. Ensure that the orchestrator is running and reachable at %s
2. Update the configuration to use a different orchestrator address using:
   a. The '-c %s=<new_address>' flag with your serve command
   b. Set the address in a configuration file with '%s config set %s=<new_address>'`,
			servers, types.ComputeOrchestratorsKey, os.Args[0], types.ComputeOrchestratorsKey)

		if servers == defaultServers {
			hint += `
3. If you are trying to connect to the demo network, use 'bootstrap.demo.bacalhau.org:4222' as your address`
		}

		return bacerrors.New("no orchestrator available for connection at %s", servers).
			WithComponent(transportClientComponent).
			WithCode(bacerrors.ConfigurationError).
			WithHint(hint)
	default:
		return bacerrors.Wrap(err, "failed to connect to %s", servers).
			WithComponent(transportClientComponent).
			WithCode(bacerrors.ConfigurationError)
	}

}
