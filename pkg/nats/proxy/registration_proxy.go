package proxy

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

type RegistrationProxyParams struct {
	Conn *nats.Conn
}

// RegistrationProxy is a proxy for a compute node to register itself with a requester node.
// The proxy can forward callbacks to a remote requester node, or locally if the node is the requester and a
// LocalCallback is provided.
type RegistrationProxy struct {
	conn *nats.Conn
}

func NewRegistrationProxy(params RegistrationProxyParams) *RegistrationProxy {
	proxy := &RegistrationProxy{
		conn: params.Conn,
	}
	return proxy
}

func (p *RegistrationProxy) Register(ctx context.Context) error {
	fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-")
	fmt.Println("Proxy Register() method called, will send message")
	fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-")
	//proxyCallbackRequest(ctx, p.conn, result.RoutingMetadata.TargetPeerID, OnBidComplete, result)

	return nil
}
