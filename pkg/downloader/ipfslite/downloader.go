package ipfslite

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/ipfs/go-bitswap/client"
	"github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-libipfs/blocks"
	"github.com/ipfs/go-libipfs/files"
	"github.com/ipfs/go-merkledag"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// See https://docs.ipfs.tech/how-to/peering-with-content-providers/#content-provider-list
var publicIPFSPeers = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
}

type Downloader struct {
	settings *model.DownloaderSettings
	cm       *system.CleanupManager
}

func NewIPFSLiteDownloader(cm *system.CleanupManager, settings *model.DownloaderSettings) *Downloader {
	return &Downloader{
		cm:       cm,
		settings: settings,
	}
}

func (ipfsDownloader *Downloader) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (ipfsDownloader *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader/ipfs.Downloader.FetchResult")
	defer span.End()

	n, err := newLiteNode(ctx)
	if err != nil {
		n.Close(ctx)
		return err
	}
	defer n.Close(ctx)

	err = func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			result.Data.Name,
			result.Data.CID, downloadPath,
		)

		innerCtx, cancel := context.WithDeadline(ctx, time.Now().Add(ipfsDownloader.settings.Timeout))
		defer cancel()

		return n.Get(innerCtx, result.Data.CID, downloadPath)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

type LiteNode struct {
	Host     host.Host
	Bistswap *client.Client
	Session  ipldformat.NodeGetter
}

// Close closes all the resources associated with the node. Must be called.
func (n LiteNode) Close(ctx context.Context) {
	if err := n.Host.Close(); err != nil {
		log.Ctx(ctx).Error().Msg("Closing host.")
	}
	if err := n.Bistswap.Close(); err != nil {
		log.Ctx(ctx).Error().Msg("Closing Bitswap client.")
	}
}

func (n LiteNode) Get(ctx context.Context, c, outputPath string) error {
	// Output path is required to not exist yet:
	ok, err := system.PathExists(outputPath)
	if err != nil {
		return err
	}
	if ok {
		return fmt.Errorf("output path '%s' already exists", outputPath)
	}

	// Get the file from the node
	dserv := merkledag.NewReadOnlyDagService(n.Session)
	nd, err := dserv.Get(ctx, cid.MustParse(c))
	if err != nil {
		return err
	}

	unixFSNode, err := unixfile.NewUnixfsFile(ctx, dserv, nd)
	if err != nil {
		return err
	}

	if err := files.WriteTo(unixFSNode, outputPath); err != nil {
		return fmt.Errorf("failed to write to '%s': %w", outputPath, err)
	}

	return nil
}

func newLiteNode(ctx context.Context) (LiteNode, error) {
	node := LiteNode{}
	// Make an ID
	privateKey, err := makeIdentity()
	if err != nil {
		return node, err
	}

	// Make a host
	node.Host, err = makeHost(0, privateKey)
	if err != nil {
		return node, err
	}

	// Connecting to public IPFS peers
	for _, addr := range publicIPFSPeers {
		err = connectToPeers(ctx, node.Host, addr)
		if err != nil {
			return node, err
		}
	}

	// The datastore for this node
	datastore := datastore.NewNullDatastore() // i.e. don't cache or store anything
	bs := blockstore.NewBlockstore(datastore)

	// Create a DHT client, which is a content routing client that uses the DHT
	dhtRouter := dht.NewDHTClient(ctx, node.Host, datastore)

	// TODO The below is only available in kubo 1.18.1, probably best to copy over...
	// Create HTTP client, which routes via contact.cid
	// privkeyb, err := crypto.MarshalPrivateKey(privateKey)
	// if err != nil {
	// 	return node, err
	// }

	// httpRouter, err := kuborouting.ConstructHTTPRouter("https://cid.contact", node.Host.ID().Pretty(), []string{"/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/udp/4001/quic"}, base64.StdEncoding.EncodeToString(privkeyb))
	// if err != nil {
	// 	return node, err
	// }

	// Create a bitswap router, which contacts various routers in parallel
	router := routinghelpers.NewComposableParallel([]*routinghelpers.ParallelRouter{
		{
			Timeout:     5 * time.Minute, // Timeouts TODO
			IgnoreError: false,
			Router:      dhtRouter,
		},
		// {
		// 	Timeout:     5 * time.Minute, // Timeouts TODO
		// 	IgnoreError: false,
		// 	Router:      httpRouter,
		// },
	})

	// Create a new bitswap network. This is the thing that actually sends and receives bitswap messages over libp2p.
	n := network.NewFromIpfsHost(node.Host, router)
	// Create a notifier to announce when a block has been received
	blockNotifier := client.WithBlockReceivedNotifier(&CustomBlockReceivedNotifier{})
	// Now create a bitswap client and start the bitswap service. This allows us to make requests.
	node.Bistswap = client.New(ctx, n, bs, blockNotifier)
	n.Start(node.Bistswap)

	// Now we can create a new block service and a DAG service, which manages block requests and navigation
	blockService := blockservice.New(bs, node.Bistswap)
	nodeGetter := merkledag.NewDAGService(blockService)
	// A DAG session ensures that if multiple blocks are requested (a directory-based CID, for example)
	// they are managed in a single request
	node.Session = merkledag.NewSession(ctx, nodeGetter)

	// This periodically prints the bitswap stats
	go periodicFunc(ctx, func() {
		log.Ctx(ctx).Debug().Bool("IsOnline", node.Bistswap.IsOnline()).Int("Wantlist", len(node.Bistswap.GetWantlist())).Int("WantHaves", len(node.Bistswap.GetWantHaves())).Int("WantBlocks", len(node.Bistswap.GetWantBlocks())).Msg("Bitswap stats")
		// for _, c := range node.Bistswap.GetWantlist() {
		// log.Printf("  %s - %d\n", c.String(), c.Type())
		// }
	})

	return node, nil
}

func makeIdentity() (crypto.PrivKey, error) {
	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func makeHost(listenPort int, privateKey crypto.PrivKey) (host.Host, error) {

	// Some basic libp2p options, see the go-libp2p docs for more details
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)), // port we are listening on, limiting to a single interface and protocol for simplicity
		libp2p.Identity(privateKey),
	}

	return libp2p.New(opts...)
}

func connectToPeers(ctx context.Context, h host.Host, targetPeer string) error {
	// Turn the targetPeer into a multiaddr.
	maddr, err := multiaddr.NewMultiaddr(targetPeer)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	// Directly connect to the peer that we know has the content
	// Generally this peer will come from whatever content routing system is provided, however go-bitswap will also
	// ask peers it is connected to for content so this will work
	if err := h.Connect(ctx, *info); err != nil {
		return err
	}
	return nil
}

type CustomBlockReceivedNotifier struct{}

func (c *CustomBlockReceivedNotifier) ReceivedBlocks(p peer.ID, blks []blocks.Block) {
	log.Printf("received %d blocks from peer %s", len(blks), p.String())
}

func periodicFunc(ctx context.Context, f func()) {
	f()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f()
		}
	}
}
