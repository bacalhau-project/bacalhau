package client

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

func (c *Client) Net() *Net {
	return &Net{client: c}
}

type Net struct {
	client *Client
}

type PeersRequest struct {
}

type PeersResponse struct {
	Peers []peer.AddrInfo
}

func (g PeersResponse) Normalize() {
	// TODO norm the norms norm norm norm
}

func (n *Net) Peers() (*PeersResponse, error) {
	var resp PeersResponse
	if err := n.client.get("/api/v1/net/peerss", &apimodels.BaseGetRequest{}, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type ConnectPeersRequest struct {
	apimodels.BaseRequest
	Peer peer.AddrInfo
}

func (c *ConnectPeersRequest) ToHTTPRequest() *apimodels.HTTPRequest {
	// TODO idk what I am supposed to do here.
	return c.BaseRequest.ToHTTPRequest()
}

type ConnectPeersResponse struct {
	Success bool
}

func (c ConnectPeersResponse) Normalize() {
	// TODO noam chomsky
}

func (n *Net) Connect(p peer.AddrInfo) (*ConnectPeersResponse, error) {
	var resp ConnectPeersResponse
	if err := n.client.post("/api/v1/net/connect", &ConnectPeersRequest{Peer: p}, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type DisconnectPeersRequest struct {
	apimodels.BaseRequest
	Peer peer.ID
}

type DisconnectPeersResponse struct {
	Success bool
}

func (c *DisconnectPeersResponse) Normalize() {
	// TODO NoRmALiZe LoVE
}

func (n *Net) Disconnect(p peer.ID) (*DisconnectPeersResponse, error) {
	var resp DisconnectPeersResponse
	if err := n.client.post("/api/v1/net/disconnect", &DisconnectPeersRequest{Peer: p}, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type PingPeerRequest struct {
	apimodels.BaseRequest
	Peer peer.ID
}

type PingPeerResponse struct {
	TTL time.Duration
	Msg string
}

func (p *PingPeerResponse) Normalize() {
	// TODO normy nomry norm norm
}

func (n *Net) Ping(p peer.ID) (*PingPeerResponse, error) {
	var resp PingPeerResponse
	if err := n.client.post("/api/v1/net/ping", &PingPeerRequest{Peer: p}, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type AddressesResponse struct {
	Addresses []multiaddr.Multiaddr
}

func (a AddressesResponse) Normalize() {
	// TODO whatever
}

func (n *Net) Addresses() (*AddressesResponse, error) {
	var resp AddressesResponse
	if err := n.client.get("/api/v1/net/addresses", &apimodels.BaseGetRequest{}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
