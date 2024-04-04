package nodefx

import (
	"context"
	"fmt"

	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
)

func NATSS(cfg nats_transport.NATSTransportConfig) (*nats_transport.NATSTransport, error) {
	// TODO(forrest): NewNATSTransport needs to be broken up into the parts that
	// create the transport and the parts that connect, i.e. the constructor does too much.
	// once that is complete use a lifecycle here to start and stop the transport.
	natsTransportLayer, err := nats_transport.NewNATSTransport(context.TODO(), cfg)
	if err != nil {
		return nil, fmt.Errorf("creating NATS transport layer: %w", err)
	}
	return natsTransportLayer, nil
}
