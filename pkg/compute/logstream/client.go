package logstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"

	"github.com/bacalhau-project/bacalhau/pkg/logger"

	ma "github.com/multiformats/go-multiaddr"
)

// 	defer stream.Close()

type LogStreamClient struct {
	host      host.Host
	stream    network.Stream
	connected bool
}

// NewLogStreamClient creates a new client communicating with the
// provided multiaddr string.
func NewLogStreamClient(ctx context.Context, address string) (*LogStreamClient, error) {
	host, err := libp2p.New([]libp2p.Option{libp2p.DisableRelay()}...)
	if err != nil {
		return nil, fmt.Errorf("logstreamclient failed to create host: %s", err)
	}

	maddr, err := ma.NewMultiaddr(address)
	if err != nil {
		return nil, fmt.Errorf("logstreamclient failed to parse address: %s", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return nil, fmt.Errorf("logstreamclient failed to create Peer: %s", err)
	}

	addresses := host.Peerstore().Addrs(info.ID)
	if len(addresses) == 0 {
		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.TempAddrTTL)
	}

	stream, err := host.NewStream(ctx, info.ID, "/bacalhau/compute/logs/1.0.0")
	if err != nil {
		return nil, fmt.Errorf("logstreamclient failed to open stream: %s", err)
	}

	return &LogStreamClient{
		host:      host,
		stream:    stream,
		connected: false,
	}, nil
}

// Connect sends the initial request to the logserver before
func (c *LogStreamClient) Connect(ctx context.Context, executionID string, withHistory bool, follow bool) error {
	if c.connected {
		return fmt.Errorf("logstream client is already connected")
	}

	streamRequest := LogStreamRequest{
		ExecutionID: executionID,
		WithHistory: withHistory,
		Follow:      follow,
	}

	err := json.NewEncoder(c.stream).Encode(streamRequest)
	if err != nil {
		return fmt.Errorf("logstream client failed to encode initial request when connecting: %s", err)
	}

	c.connected = true

	return nil
}

// Close will close the underlying stream and resources in-use.
func (c *LogStreamClient) Close() {
	if !c.connected {
		return
	}

	c.connected = false
	c.host.Close()
	c.stream.Close()
}

// ReadDataFrame reads a single dataframe from the client's stream (if connected)
func (c *LogStreamClient) ReadDataFrame(ctx context.Context) (logger.DataFrame, error) {
	if !c.connected {
		return logger.EmptyDataFrame, fmt.Errorf("logstream client connection state is %t", c.connected)
	}

	frame, err := logger.NewDataFrameFromReader(c.stream)
	if err == io.EOF {
		return logger.EmptyDataFrame, fmt.Errorf("logstreamclient connection closed by peer: %s", err)
	}

	if err != nil {
		return logger.EmptyDataFrame, fmt.Errorf("logstreamclient error reading dataframe: %s", err)
	}

	return frame, nil
}
