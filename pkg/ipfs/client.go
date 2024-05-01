package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/ipfs/boxo/files"
	icorepath "github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	httpapi "github.com/ipfs/kubo/client/rpc"
	icore "github.com/ipfs/kubo/core/coreiface"
	icoreoptions "github.com/ipfs/kubo/core/coreiface/options"
	ipfsopts "github.com/ipfs/kubo/core/coreiface/options"
	ma "github.com/multiformats/go-multiaddr"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func CIDFromStr(str string) (cid.Cid, error) {
	c, err := cid.Decode(str)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to decode CID from %q: %w", str, err)
	}
	return c, nil
}

// WriteTo is a helper method that write the result of a call to Node.Get to a path.
func WriteTo(n files.Node, path string) error {
	return files.WriteTo(n, path)
}

type Node interface {
	ID(ctx context.Context) (string, error)
	APIAddress() string
	Get(ctx context.Context, c cid.Cid) (files.Node, error)
	Size(ctx context.Context, c cid.Cid) (uint64, error)
	Has(ctx context.Context, c cid.Cid) (bool, error)
	Put(ctx context.Context, path string) (cid.Cid, error)
	GetTreeNode(ctx context.Context, cid cid.Cid) (IPLDTreeNode, error)
}

var _ Node = (*client)(nil)

type client struct {
	API  icore.CoreAPI
	addr string
}

func New(apiAddr string) (*client, error) {
	addr, err := ma.NewMultiaddr(apiAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse api address '%s': %w", apiAddr, err)
	}

	// This http.Transport is the same that httpapi.NewApi would use if we weren't passing in our own http.Client
	defaultTransport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
	}
	api, err := httpapi.NewApiWithClient(addr, &http.Client{
		Transport: otelhttp.NewTransport(
			defaultTransport,
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
			otelhttp.WithSpanOptions(trace.WithAttributes(semconv.PeerService("ipfs"))),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", apiAddr, err)
	}

	client := &client{
		API:  api,
		addr: apiAddr,
	}
	return client, nil
}

func (c *client) ID(ctx context.Context) (string, error) {
	key, err := c.API.Key().Self(ctx)
	if err != nil {
		return "", err
	}
	return key.ID().String(), nil
}

func (c *client) APIAddress() string {
	return c.addr
}

func (c *client) Get(ctx context.Context, cid cid.Cid) (files.Node, error) {
	path := icorepath.FromCid(cid)
	node, err := c.API.Unixfs().Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get ipfs cid '%s': %w", cid, err)
	}

	return node, nil
}

func (c *client) Put(ctx context.Context, path string) (cid.Cid, error) {
	st, err := os.Stat(path)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to stat file '%s': %w", path, err)
	}

	node, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to create ipfs node: %w", err)
	}

	// Pin uploaded file/directory to local storage to prevent deletion by GC.
	addOptions := []icoreoptions.UnixfsAddOption{
		icoreoptions.Unixfs.Pin(true),
	}

	ipfsPath, err := c.API.Unixfs().Add(ctx, node, addOptions...)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to add file '%s': %w", path, err)
	}

	return ipfsPath.RootCid(), nil
}

func (c *client) Size(ctx context.Context, cid cid.Cid) (uint64, error) {
	path := icorepath.FromCid(cid)

	node, err := c.API.ResolveNode(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve node '%s': %w", cid, err)
	}

	return node.Size()
}

func (c *client) Has(ctx context.Context, cid cid.Cid) (bool, error) {
	// TODO(forrest) [correctness]: I am not 100% sure this is right, but here
	// we create an offline api and then attempt to retrieve content from it
	// if the node has the content locally this will succeed, if it doesn't this will
	// fail.
	offlineAPI, err := c.API.WithOptions(func(settings *ipfsopts.ApiSettings) error {
		settings.Offline = true
		settings.FetchBlocks = false
		return nil
	})
	if err != nil {
		return false, err
	}
	_, err = offlineAPI.Dag().Get(ctx, cid)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func (c *client) GetTreeNode(ctx context.Context, cid cid.Cid) (IPLDTreeNode, error) {
	path := icorepath.FromCid(cid)

	ipldNode, err := c.API.ResolveNode(ctx, path)
	if err != nil {
		return IPLDTreeNode{}, fmt.Errorf("failed to resolve node '%s': %w", cid, err)
	}

	return getTreeNode(ctx, ipld.NewNavigableIPLDNode(ipldNode, c.API.Dag()), []string{})
}
