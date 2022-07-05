package ipfs

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/system"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// Client is a front-end for an ipfs node's API endpoints. You can create
// Client instances manually by connecting to an ipfs node's API multiaddr,
// or automatically from an active Node instance.
type Client struct {
	api  icore.CoreAPI
	addr string
}

// NewClient creates an API client for the given ipfs node API multiaddress.
// NOTE: the API address is _not_ the same as the swarm address
func NewClient(apiAddr string) (*Client, error) {
	addr, err := ma.NewMultiaddr(apiAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse api address '%s': %w", apiAddr, err)
	}

	api, err := httpapi.NewApi(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", apiAddr, err)
	}

	log.Debug().Msgf("Created IPFS client for node API address: %s", apiAddr)
	return &Client{
		api:  api,
		addr: apiAddr,
	}, nil
}

// WaitUntilAvailable blocks the current goroutine until the client is able
// to successfully make requests to the server. Useful for setting up local
// test networks. WaitUntilAvailable will respect context deadlines/cancels,
// and will propagate context cancellations back to the caller.
// NOTE: if you do not pass a context with a deadline/cancel in to this
//       function, it may attempt to call the api server forever.
func (cl *Client) WaitUntilAvailable(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		_, err := cl.ID(ctx)
		if err != nil {
			log.Debug().Msgf("non-critical error waiting for node availability: %v", err)
		} else {
			return nil
		}

		// don't spin as fast as possible:
		time.Sleep(time.Second)
	}
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

// APIAddress returns api address that was used to connect to the node.
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

// Get fetches a file or directory from the ipfs network.
func (cl *Client) Get(ctx context.Context, cid, outputPath string) error {
	node, err := cl.api.Unixfs().Get(ctx, path.New(cid))
	if err != nil {
		return fmt.Errorf("failed to get file '%s': %w", cid, err)
	}

	return files.WriteTo(node, outputPath)
}

// Put uploads a file or directory to the ipfs network.
func (cl *Client) Put(ctx context.Context, inputPath string) (string, error) {
	st, err := os.Stat(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file '%s': %w", inputPath, err)
	}

	node, err := files.NewSerialFile(inputPath, false, st)
	if err != nil {
		return "", fmt.Errorf("failed to create ipfs node: %w", err)
	}

	cid, err := cl.api.Unixfs().Add(ctx, node)
	if err != nil {
		return "", fmt.Errorf("failed to add file '%s': %w", inputPath, err)
	}

	return cid.String(), nil
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
		return false, fmt.Errorf("error fetching node's ipfs id: %w", err)
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
