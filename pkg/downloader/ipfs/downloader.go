package ipfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/rs/zerolog/log"

	bac_config "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	ipfssource "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Downloader struct {
	cm   *system.CleanupManager
	node *ipfs.Node // defaults to nil
}

func NewIPFSDownloader(cm *system.CleanupManager) *Downloader {
	return &Downloader{
		cm: cm,
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

func (d *Downloader) describeResult(ctx context.Context, result ipfssource.Source) (map[string]string, error) {
	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	ipfsClient, err := d.getClient(ctx)

	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("cid", result.CID).
		Msg("Describing contents of result CID")

	tree, err := ipfsClient.GetTreeNode(ctx, result.CID)
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

func (d *Downloader) FetchResult(ctx context.Context, item downloader.DownloadItem) (string, error) {
	sourceSpec, err := ipfssource.DecodeSpec(item.Result)
	if err != nil {
		return "", err
	}

	cid := sourceSpec.CID
	resultPath := filepath.Join(item.ParentPath, cid)
	downloadPath := resultPath

	// If we're downloading a single file, we need to find the CID of that file,
	if item.SingleFile != "" {
		filemap, err := d.describeResult(ctx, sourceSpec)
		if err != nil {
			return "", err
		}
		fileCID, present := filemap[item.SingleFile]
		if !present {
			return "", fmt.Errorf("failed to find cid for %s", item.SingleFile)
		}
		cid = fileCID
		downloadPath = filepath.Join(resultPath, item.SingleFile)
		err = os.MkdirAll(filepath.Dir(downloadPath), downloader.DownloadFolderPerm)
		if err != nil {
			return "", err
		}
	}

	alreadyExists, err := downloader.IsAlreadyDownloaded(downloadPath)
	if err != nil {
		return "", err
	}
	if alreadyExists {
		// We don't want to download the same CID twice
		log.Ctx(ctx).Debug().
			Str("CID", cid).
			Msg("asked to download a CID a second time")
		return resultPath, nil
	}

	ipfsClient, err := d.getClient(ctx)
	if err != nil {
		return "", err
	}

	log.Ctx(ctx).Debug().
		Str("cid", cid).
		Str("path", downloadPath).
		Msg("Downloading result CID")

	err = ipfsClient.Get(ctx, cid, downloadPath)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result")
		}

		return "", err
	}
	// we always return the path of the result cid, even if it's a single file
	return resultPath, nil
}
