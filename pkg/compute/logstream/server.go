package logstream

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog/log"
)

const (
	ExecutionLogsID = "/bacalhau/compute/logs/1.0.0"
)

type LogStreamServer struct {
	Address string
	host    host.Host
	ctx     context.Context
}

func NewLogStreamServer(ctx context.Context, host host.Host) *LogStreamServer {
	log.Ctx(ctx).Debug().Msg("Creating new LogStreamServer")

	svr := &LogStreamServer{
		ctx:  util.NewDetachedContext(ctx),
		host: host,
	}
	svr.Address = findTCPAddress(host)

	host.SetStreamHandler(ExecutionLogsID, svr.Handle)
	return svr
}

func findTCPAddress(host host.Host) string {
	peerID := host.ID().Pretty()
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID))

	for _, addr := range host.Addrs() {
		for _, protocol := range addr.Protocols() {
			if protocol.Name == "tcp" {
				return addr.Encapsulate(hostAddr).String()
			}
		}
	}

	// If we can't find TCP, then we'll go with the first record
	addr := host.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

func (s *LogStreamServer) Handle(stream network.Stream) {
	log.Ctx(s.ctx).Debug().Msg("Handling new logging request")

	// TODO: Read Header/Request
	// TODO: Connect to execution and get streams
	// TODO: Use the multistream to write back the output

	_, _ = stream.Write([]byte("Hello"))
}
