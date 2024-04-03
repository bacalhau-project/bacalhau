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

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	ipfssource "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Downloader struct {
	cm     *system.CleanupManager
	client *ipfs.Client
}

func NewIPFSDownloader(cm *system.CleanupManager, client *ipfs.Client) *Downloader {
	return &Downloader{
		cm:     cm,
		client: client,
	}
}

func (d *Downloader) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (d *Downloader) describeResult(ctx context.Context, result ipfssource.Source) (map[string]string, error) {
	log.Ctx(ctx).Debug().
		Str("cid", result.CID).
		Msg("Describing contents of result CID")

	tree, err := d.client.GetTreeNode(ctx, result.CID)
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

	log.Ctx(ctx).Debug().
		Str("cid", cid).
		Str("path", downloadPath).
		Msg("Downloading result CID")

	err = d.client.Get(ctx, cid, downloadPath)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result")
		}

		return "", err
	}
	// we always return the path of the result cid, even if it's a single file
	return resultPath, nil
}
