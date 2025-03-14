package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/ipfs/boxo/files"
	dag "github.com/ipfs/boxo/ipld/merkledag"
	ft "github.com/ipfs/boxo/ipld/unixfs"
	icorepath "github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	httpapi "github.com/ipfs/kubo/client/rpc"
	icore "github.com/ipfs/kubo/core/coreiface"
	icoreoptions "github.com/ipfs/kubo/core/coreiface/options"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// Client is a front-end for an ipfs node's API endpoints
type Client struct {
	API  icore.CoreAPI
	addr string
}

// NewClient creates an API client for the given ipfs node API multiaddress.
func NewClient(ctx context.Context, apiAddr string) (*Client, error) {
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
		Transport: defaultTransport,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", apiAddr, err)
	}

	client := &Client{
		API:  api,
		addr: apiAddr,
	}

	id, err := client.ID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", apiAddr, err)
	}
	log.Ctx(ctx).Debug().Msgf("Created remote IPFS client for node API address: %s, with id: %s", apiAddr, id)
	return client, nil
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

	addrs = lo.Map(addrs, func(f ma.Multiaddr, _ int) ma.Multiaddr {
		return f.Encapsulate(p2pID)
	})

	return addrs, nil
}

// SwarmAddresses returns a list of swarm addresses the node has announced.
func (cl Client) SwarmAddresses(ctx context.Context) ([]string, error) {
	multiAddresses, err := cl.SwarmMultiAddresses(ctx)
	if err != nil {
		return nil, err
	}

	if len(multiAddresses) == 0 {
		return nil, fmt.Errorf("no swarm addresses found")
	}

	// It's common for callers to this function to use the result to connect to another IPFS node.
	// This sorts the addresses so IPv4 localhost is first, with the aim of using the localhost connection during tests
	// and so avoid any unneeded network hops. Other callers to this either sort the list themselves or just output the
	// full list.
	multiAddresses = SortLocalhostFirst(multiAddresses)

	addresses := lo.Map(multiAddresses, func(f ma.Multiaddr, _ int) string {
		return f.String()
	})

	return addresses, nil
}

// SwarmConnect establishes concurrent connections to each peer from the provided `peers` list.
// It spawns a goroutine for each peer connection. In the event of a connection failure,
// a warning log containing the error and peer details is generated.
func (cl Client) SwarmConnect(ctx context.Context, peers []peer.AddrInfo) {
	var wg sync.WaitGroup
	for _, p := range peers {
		wg.Add(1)
		go func(ctx context.Context, p peer.AddrInfo) {
			defer wg.Done()
			if err := cl.API.Swarm().Connect(ctx, p); err != nil {
				log.Ctx(ctx).Warn().Err(err).Stringer("peer", p).Msg("failed to connect to peer")
			}
		}(ctx, p)
	}
	wg.Wait()
}

// Get fetches a file or directory from the ipfs network.
func (cl Client) Get(ctx context.Context, cid, outputPath string) error {
	// Output path is required to not exist yet:
	ok, err := system.PathExists(outputPath)
	if err != nil {
		return fmt.Errorf("unable to check if path %s exists: %w", outputPath, err)
	}
	if ok {
		return fmt.Errorf("output path '%s' already exists", outputPath)
	}

	path, err := pathFromCIDString(cid)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to create path from CID: %s", cid))
	}

	node, err := cl.API.Unixfs().Get(ctx, path)
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

	cid := ipfsPath.RootCid().String()
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
	path, err := pathFromCIDString(cid)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to create path from CID in call to Stat(): %s", cid))
	}

	node, err := cl.API.ResolveNode(ctx, path)
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

func (cl Client) GetCidSize(ctx context.Context, cidStr string) (uint64, error) {
	c, err := cid.Decode(cidStr)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("unable to decode CID in call to GetCidSize(): %s", cidStr))
	}

	content, err := cl.API.Dag().Get(ctx, c)
	if err != nil {
		return 0, fmt.Errorf("unable to retrieve cid %s in call to GetCidSize(): %w", c, err)
	}

	size, err := content.Size()
	if err != nil {
		return 0, fmt.Errorf("unable to determine cid %s size GetCidSize(): %w", c, err)
	}

	return size, nil
}

// HasCID returns true if the node has the given CID locally, whether pinned or not.
func (cl Client) HasCID(ctx context.Context, cidStr string) (bool, error) {
	// create an offline API that will not search the network for content.
	offlineAPI, err := cl.API.WithOptions(
		icoreoptions.Api.FetchBlocks(false),
		icoreoptions.Api.Offline(true),
	)
	if err != nil {
		return false, err
	}
	c, err := cid.Decode(cidStr)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("unable to decode CID: %s", cidStr))
	}
	// attempt to stat the block in the local IPFS, if it's not found w/ the offlineAPI then the content
	// is not local to the IPFS node.
	_, err = offlineAPI.Dag().Get(ctx, c)
	if err != nil {
		if ipld.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// stating the CID was successful, IPFS has this content locally.
	return true, nil
}

func (cl Client) GetTreeNode(ctx context.Context, cid string) (IPLDTreeNode, error) {
	path, err := pathFromCIDString(cid)
	if err != nil {
		return IPLDTreeNode{}, errors.Wrap(err, fmt.Sprintf("unable to create path from CID: %s", cid))
	}

	ipldNode, err := cl.API.ResolveNode(ctx, path)
	if err != nil {
		return IPLDTreeNode{}, fmt.Errorf("failed to resolve node '%s': %w", cid, err)
	}

	return getTreeNode(ctx, ipld.NewNavigableIPLDNode(ipldNode, cl.API.Dag()), []string{})
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

func pathFromCIDString(cidString string) (icorepath.Path, error) {
	cid, err := cid.Decode(cidString)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to decode CID: %s", cidString))
	}

	return icorepath.FromCid(cid), nil
}
