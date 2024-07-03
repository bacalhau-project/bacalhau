package nats

import (
	"context"

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
		return nil, err
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
