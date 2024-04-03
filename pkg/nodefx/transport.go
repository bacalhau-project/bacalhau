package nodefx

import (
	"context"
	"fmt"

	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
)

func NATSS(cfg *NodeConfig) (*nats_transport.NATSTransport, error) {
	natsTransportLayer, err := nats_transport.NewNATSTransport(context.TODO(), *cfg.TransportConfig)
	if err != nil {
		return nil, fmt.Errorf("creating NATS transport layer: %w", err)
	}
	return natsTransportLayer, nil
}
