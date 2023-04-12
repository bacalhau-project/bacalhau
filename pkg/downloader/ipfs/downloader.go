package ipfs

import (
	"context"
	"errors"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
)

type Downloader struct {
	settings *model.DownloaderSettings
	cm       *system.CleanupManager
}

func NewIPFSDownloader(cm *system.CleanupManager, settings *model.DownloaderSettings) *Downloader {
	return &Downloader{
		cm:       cm,
		settings: settings,
	}
}

func (d *Downloader) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (d *Downloader) createTemporaryNode(ctx context.Context) (ipfs.Client, *ipfs.Node, error) {
	log.Ctx(ctx).Debug().Msg("creating ipfs node")

	newNode := ipfs.NewNode
	if d.settings.LocalIPFS {
		newNode = ipfs.NewLocalNode
	}

	node, err := newNode(ctx, d.cm, strings.Split(d.settings.IPFSSwarmAddrs, ","))
	if err != nil {
		return ipfs.Client{}, nil, err
	}

	return node.Client(), node, nil
}

func (d *Downloader) DescribeResult(ctx context.Context, result model.PublishedResult) (map[string]string, error) {
	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	ipfsClient, node, err := d.createTemporaryNode(ctx)
	defer closer.ContextCloserWithLogOnError(ctx, "IPFS node", node)

	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("cid", result.Data.CID).
		Str("name", result.Data.Name).
		Msg("Describing contents of result CID")

	tree, err := ipfsClient.GetTreeNode(ctx, result.Data.CID)
	if err != nil {
		return nil, err
	}

	files := make(map[string]string)

	nodes, err := ipfs.FlattenTreeNode(ctx, tree)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if len(node.Path) > 0 {
			files[node.Path[0]] = node.Cid.String()
		}
	}

	return files, nil
}

func (d *Downloader) FetchResult(ctx context.Context, item model.DownloadItem) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader/ipfs.Downloader.FetchResult")
	defer span.End()

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	ipfsClient, node, err := d.createTemporaryNode(ctx)
	defer closer.ContextCloserWithLogOnError(ctx, "IPFS node", node)

	if err != nil {
		return err
	}

	err = func() error {
		log.Ctx(ctx).Debug().
			Str("cid", item.Identifier).
			Str("name", item.Name).
			Str("path", item.Target).
			Msg("Downloading result CID")

		innerCtx, cancel := context.WithTimeout(ctx, d.settings.Timeout)
		defer cancel()

		return ipfsClient.Get(innerCtx, item.Identifier, item.Target)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result")
		}

		return err
	}
	return nil
}
