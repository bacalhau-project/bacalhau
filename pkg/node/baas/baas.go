package baas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ProtocolID = "/bacalhau/baas/0.0.1"

type RequestMessage struct {
	Key       string
	Addresses []string
}

type ResponseMessage struct {
	Peers []Peer
}

type Peer struct {
	PeerID    string
	Addresses []string
}

func NewService(h host.Host, key string) *Service {
	return &Service{h: h, key: key}
}

type Service struct {
	h   host.Host
	key string
}

func (s *Service) DoTheThing(ctx context.Context, pid peer.ID) (*ResponseMessage, error) {
	strm, err := s.h.NewStream(ctx, pid, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("opening new stream: %w", err)
	}

	req := &RequestMessage{
		Key:       s.key,
		Addresses: []string{"test_address_from_bacalhau"},
	}
	// send the details over
	if err := json.NewEncoder(strm).Encode(req); err != nil {
		return nil, fmt.Errorf("writing rpc to peer: %w", err)
	}

	var rsp ResponseMessage
	if err := json.NewDecoder(strm).Decode(&rsp); err != nil {
		return nil, fmt.Errorf("reading rpc from peer: %w", err)
	}

	// TODO close the stream.

	return &rsp, nil
}
