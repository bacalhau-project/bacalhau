package ipfs

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/generic"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	ft "github.com/ipfs/go-unixfs"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icoreoptions "github.com/ipfs/interface-go-ipfs-core/options"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

// Client is a front-end for an ipfs node's API endpoints. You can create
// Client instances manually by connecting to an ipfs node's API multiaddr,
// or automatically from an active Node instance.
type Client struct {
	API  icore.CoreAPI
	addr string
}

// NewClientUsingRemoteHandler creates an API client for the given ipfs node API multiaddress.
// NOTE: the API address is _not_ the same as the swarm address
func NewClientUsingRemoteHandler(apiAddr string) (Client, error) {
	addr, err := ma.NewMultiaddr(apiAddr)
	if err != nil {
		return Client{}, fmt.Errorf("failed to parse api address '%s': %w", apiAddr, err)
	}

	api, err := httpapi.NewApi(addr)
	if err != nil {
		return Client{}, fmt.Errorf("failed to connect to '%s': %w", apiAddr, err)
	}

	client := Client{
		API:  api,
		addr: apiAddr,
	}

	id, err := client.ID(context.Background())
	if err != nil {
		return Client{}, fmt.Errorf("failed to connect to '%s': %w", apiAddr, err)
	}
	log.Debug().Msgf("Created remote IPFS client for node API address: %s, with id: %s", apiAddr, id)
	return client, nil
}

const MagicInternalIPFSAddress = "memory://in-memory-node/"

func NewClient(api icore.CoreAPI) Client {
	return Client{
		API:  api,
		addr: MagicInternalIPFSAddress,
	}
}

// WaitUntilAvailable blocks the current goroutine until the client is able
// to successfully make requests to the server. Useful for setting up local
// test networks. WaitUntilAvailable will respect context deadlines/cancels,
// and will propagate context cancellations back to the caller.
// NOTE: if you do not pass a context with a deadline/cancel in to this
//
//	function, it may attempt to call the api server forever.
func (cl Client) WaitUntilAvailable(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		_, err := cl.ID(ctx)
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("non-critical error waiting for node availability: %v", err)
		} else {
			return nil
		}

		// don't spin as fast as possible:
		time.Sleep(time.Second)
	}
}

// ID returns the node's ipfs ID.
func (cl Client) ID(ctx context.Context) (string, error) {
	key, err := cl.API.Key().Self(ctx)
	if err != nil {
		return "", err
	}

	return key.ID().String(), nil
}

// APIAddress returns Api address that was used to connect to the node.
func (cl Client) APIAddress() string {
	return cl.addr
}

func (cl Client) SwarmMultiAddresses(ctx context.Context) ([]ma.Multiaddr, error) {
	id, err := cl.API.Key().Self(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching node's ipfs id: %w", err)
	}

	p2pID, err := ma.NewMultiaddr("/p2p/" + id.ID().String())
	if err != nil {
		return nil, err
	}

	addrs, err := cl.API.Swarm().LocalAddrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching node's swarm addresses: %w", err)
	}

	addrs = generic.Map(addrs, func(f ma.Multiaddr) ma.Multiaddr {
		return f.Encapsulate(p2pID)
	})

	return addrs, nil
}

// SwarmAddresses returns a list of swarm addresses the node has announced.
func (cl Client) SwarmAddresses(ctx context.Context) ([]string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.SwarmAddresses")
	defer span.End()

	multiAddresses, err := cl.SwarmMultiAddresses(ctx)
	if err != nil {
		return nil, err
	}

	addresses := generic.Map(multiAddresses, func(f ma.Multiaddr) string {
		return f.String()
	})

	return addresses, nil
}

// Get fetches a file or directory from the ipfs network.
func (cl Client) Get(ctx context.Context, cid, outputPath string) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.Get")
	defer span.End()

	// Output path is required to not exist yet:
	ok, err := system.PathExists(outputPath)
	if err != nil {
		return err
	}
	if ok {
		return fmt.Errorf("output path '%s' already exists", outputPath)
	}

	node, err := cl.API.Unixfs().Get(ctx, icorepath.New(cid))
	if err != nil {
		return fmt.Errorf("failed to get ipfs cid '%s': %w", cid, err)
	}

	if err := files.WriteTo(node, outputPath); err != nil {
		return fmt.Errorf("failed to write to '%s': %w", outputPath, err)
	}

	return nil
}

// Put uploads and pins a file or directory to the ipfs network. Timeouts and
// cancellation should be handled by passing an appropriate context value.
func (cl Client) Put(ctx context.Context, inputPath string) (string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.Put")
	defer span.End()

	st, err := os.Stat(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file '%s': %w", inputPath, err)
	}

	node, err := files.NewSerialFile(inputPath, false, st)
	if err != nil {
		return "", fmt.Errorf("failed to create ipfs node: %w", err)
	}

	// Pin uploaded file/directory to local storage to prevent deletion by GC.
	addOptions := []icoreoptions.UnixfsAddOption{
		icoreoptions.Unixfs.Pin(true),
	}

	ipfsPath, err := cl.API.Unixfs().Add(ctx, node, addOptions...)
	if err != nil {
		return "", fmt.Errorf("failed to add file '%s': %w", inputPath, err)
	}

	cid := ipfsPath.Cid().String()
	return cid, nil
}

type IPLDType int

const (
	IPLDUnknown IPLDType = iota
	IPLDFile
	IPLDDirectory
)

type StatResult struct {
	Type IPLDType
}

// Stat returns information about an IPLD CID on the ipfs network.
func (cl Client) Stat(ctx context.Context, cid string) (*StatResult, error) {
	ctx, span := system.GetTracer().Start(ctx, "kg/ipfs.Stat")
	defer span.End()

	node, err := cl.API.ResolveNode(ctx, icorepath.New(cid))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve node '%s': %w", cid, err)
	}

	nodeType, err := getNodeType(node)
	if err != nil {
		return nil, fmt.Errorf("failed to get node type: %w", err)
	}

	return &StatResult{
		Type: nodeType,
	}, nil
}

func (cl Client) GetCidSize(ctx context.Context, cid string) (uint64, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.GetCidSize")
	defer span.End()

	stat, err := cl.API.Object().Stat(ctx, icorepath.New(cid))
	if err != nil {
		return 0, err
	}

	return uint64(stat.CumulativeSize), nil
}

// NodesWithCID returns the ipfs ids of nodes that have the given CID pinned.
func (cl Client) NodesWithCID(ctx context.Context, cid string) ([]string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.NodesWithCID")
	defer span.End()

	ch, err := cl.API.Dht().FindProviders(ctx, icorepath.New(cid))
	if err != nil {
		return nil, fmt.Errorf("error finding providers of '%s': %w", cid, err)
	}

	var res []string
	for info := range ch {
		res = append(res, info.ID.String())
	}

	return res, nil
}

// HasCID returns true if the node has the given CID locally, whether pinned or not.
func (cl Client) HasCID(ctx context.Context, cid string) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.HasCID")
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

func (cl Client) GetTreeNode(ctx context.Context, cid string) (IPLDTreeNode, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.GetTreeNode")
	defer span.End()

	ipldNode, err := cl.API.ResolveNode(ctx, icorepath.New(cid))
	if err != nil {
		return IPLDTreeNode{}, fmt.Errorf("failed to resolve node '%s': %w", cid, err)
	}

	return GetTreeNode(ctx, ipld.NewNavigableIPLDNode(ipldNode, cl.API.Dag()), []string{})
}

func getNodeType(node ipld.Node) (IPLDType, error) {
	// Taken from go-ipfs/core/commands/files.go:
	var nodeType IPLDType
	switch n := node.(type) {
	case *dag.ProtoNode:
		d, err := ft.FSNodeFromBytes(n.Data())
		if err != nil {
			return IPLDUnknown, err
		}

		switch d.Type() {
		case ft.TDirectory, ft.THAMTShard:
			nodeType = IPLDDirectory
		case ft.TFile, ft.TMetadata, ft.TRaw:
			nodeType = IPLDFile
		default:
			return IPLDUnknown, fmt.Errorf("unrecognized node type: %s", d.Type())
		}
	case *dag.RawNode:
		nodeType = IPLDFile
	default:
		return IPLDUnknown, fmt.Errorf("unrecognized node type: %T", node)
	}

	return nodeType, nil
}
