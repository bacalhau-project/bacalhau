package api

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p/core/host"
	inet "github.com/libp2p/go-libp2p/core/network"

	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
)

const ProtocolID = "/bacalhau/baas/0.0.1"

type ResponseMessage struct {
	Key       string
	Addresses []string
}

type RequestMessage struct {
	Peers []peer
}

func NewHost(port int) (host.Host, error) {
	prvKey, err := libp2p.GeneratePrivateKey(2048)
	if err != nil {
		return nil, err
	}

	h, err := libp2p.NewHost(port, prvKey)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func NewService(h host.Host, api *API) *Service {
	return &Service{
		h:   h,
		api: api,
	}
}

type Service struct {
	h   host.Host
	api *API
}

func (s *Service) HandleStream(ps inet.Stream) {
	var rsp ResponseMessage
	if err := json.NewDecoder(ps).Decode(&rsp); err != nil {
		panic(err)
	}

	if err := s.api.RegisterNode(registerNodeRequest{
		Key:       rsp.Key,
		PeerID:    ps.Conn().RemotePeer().String(),
		Addresses: append(rsp.Addresses, ps.Conn().RemoteMultiaddr().String()),
	}); err != nil {
		panic(err)
	}

	// TODO we need to make sure we don't return the node its own data, right now will will.
	peers, err := s.api.FindPeers(findPeersRequest{
		Key: rsp.Key,
	})
	if err != nil {
		panic(err)
	}

	if err := json.NewEncoder(ps).Encode(RequestMessage{Peers: peers}); err != nil {
		panic(err)
	}

	return
}
