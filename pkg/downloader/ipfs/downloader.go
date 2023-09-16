package ipfs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	bac_config "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Downloader struct {
	settings *model.DownloaderSettings
	cm       *system.CleanupManager
	node     *ipfs.Node // defaults to nil
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

func (d *Downloader) getClient(ctx context.Context) (ipfs.Client, error) {
	var cfg types.IpfsConfig
	if err := bac_config.ForKey(types.NodeIPFS, &cfg); err != nil {
		return ipfs.Client{}, err
	}

	if cfg.Connect != "" {
		log.Ctx(ctx).Debug().Msg("creating ipfs client")
		client, err := ipfs.NewClientUsingRemoteHandler(ctx, cfg.Connect)
		if err != nil {
			return ipfs.Client{}, fmt.Errorf("error creating IPFS client: %s", err)
		}

		if len(cfg.SwarmAddresses) != 0 {
			maddrs, err := ipfs.ParsePeersString(cfg.SwarmAddresses)
			if err != nil {
				return ipfs.Client{}, err
			}
			client.SwarmConnect(ctx, maddrs)
		}
		return client, nil
	}

	log.Ctx(ctx).Debug().Msg("creating ipfs node")
	if d.node == nil {
		node, err := ipfs.NewNodeWithConfig(ctx, d.cm, cfg)
		if err != nil {
			return ipfs.Client{}, err
		}

		d.node = node
	}

	return d.node.Client(), nil
}

func (d *Downloader) DescribeResult(ctx context.Context, result model.PublishedResult) (map[string]string, error) {
	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	ipfsClient, err := d.getClient(ctx)

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
			p := strings.Join(node.Path, "/")
			files[p] = node.Cid.String()
		}
	}

	return files, nil
}

func (d *Downloader) FetchResult(ctx context.Context, item model.DownloadItem) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader/ipfs.Downloader.FetchResult")
	defer span.End()

	ipfsClient, err := d.getClient(ctx)
	if err != nil {
		return err
	}

	err = func() error {
		log.Ctx(ctx).Debug().
			Str("cid", item.CID).
			Str("name", item.Name).
			Str("path", item.Target).
			Msg("Downloading result CID")

		innerCtx, cancel := context.WithTimeout(ctx, d.settings.Timeout)
		defer cancel()

		return ipfsClient.Get(innerCtx, item.CID, item.Target)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result")
		}

		return err
	}
	return nil
}
