package ipfs

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/system"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
	"go.opentelemetry.io/otel/trace"
)

// Client is a front-end for an ipfs node's API endpoints. You can create
// Client instances manually by connecting to an ipfs node's API multiaddr,
// or automatically from an active Node instance.
type Client struct {
	api  *httpapi.HttpApi
	addr string
}

func NewClient(apiAddr string) (*Client, error) {
	addr, err := ma.NewMultiaddr(apiAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse api address '%s': %w", apiAddr, err)
	}

	api, err := httpapi.NewApi(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", apiAddr, err)
	}

	return &Client{
		api:  api,
		addr: apiAddr,
	}, nil
}

// ID returns the node's ipfs ID.
func (cl *Client) ID(ctx context.Context) (string, error) {
	ctx, span := newSpan(ctx, "ID")
	defer span.End()

	key, err := cl.api.Key().Self(ctx)
	if err != nil {
		return "", err
	}

	return key.ID().String(), nil
}

// APIAddress returns the api address of the node.
func (cl *Client) APIAddress() string {
	return cl.addr
}

// SwarmAddresses returns a list of swarm addresses the node has announced.
func (cl *Client) SwarmAddresses(ctx context.Context) ([]string, error) {
	ctx, span := newSpan(ctx, "SwarmAddresses")
	defer span.End()

	id, err := cl.ID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching node's ipfs id: %w", err)
	}

	addrs, err := cl.api.Swarm().LocalAddrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching node's swarm addresses: %w", err)
	}

	var res []string
	for _, addr := range addrs {
		res = append(res, fmt.Sprintf("%s/p2p/%s", addr.String(), id))
	}

	return res, nil
}

// NodesWithCID returns the ipfs ids of nodes that have the given CID pinned.
func (cl *Client) NodesWithCID(ctx context.Context, cid string) ([]string, error) {
	ctx, span := newSpan(ctx, "NodesWithCID")
	defer span.End()

	ch, err := cl.api.Dht().FindProviders(ctx, path.New(cid))
	if err != nil {
		return nil, fmt.Errorf("error finding providers of '%s': %w", cid, err)
	}

	var res []string
	for info := range ch {
		res = append(res, info.ID.String())
	}

	return res, nil
}

// HadCID returns true if the node has the given CID pinned.
func (cl *Client) HasCID(ctx context.Context, cid string) (bool, error) {
	ctx, span := newSpan(ctx, "HasCID")
	defer span.End()

	id, err := cl.ID(ctx)
	if err != nil {
		return false, fmt.Errorf("error fetching node's ipfs id", err)
	}

	nodes, err := cl.NodesWithCID(ctx, cid)
	if err != nil {
		return false, fmt.Errorf("error fetching nodes with cid '%s': %w", cid, err)
	}

	for _, node := range nodes {
		if node == id {
			return true, nil
		}
	}

	return false, nil
}

func (cl *Client) DownloadTar(ctx context.Context, targetDir, cid string) error {
	return fmt.Errorf("not implemented: DownloadTar")
}

// TODO: #291 we need to work out how to upload a tar file
// using just the HTTP api and not needing to shell out
func (cl *Client) UploadTar(ctx context.Context, sourceDir string) (string, error) {
	return "", fmt.Errorf("not implemented: UploadTar")
}

func newSpan(ctx context.Context, api string) (context.Context, trace.Span) {
	return system.Span(ctx, "ipfs/http", api)
}
