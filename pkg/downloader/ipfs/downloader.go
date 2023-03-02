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

func (d *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader/ipfs.Downloader.FetchResult")
	defer span.End()

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.

	log.Ctx(ctx).Debug().Msg("Spinning up IPFS node")

	newNode := ipfs.NewNode
	if d.settings.LocalIPFS {
		newNode = ipfs.NewLocalNode
	}
	n, err := newNode(ctx, d.cm, strings.Split(d.settings.IPFSSwarmAddrs, ","))
	if err != nil {
		return err
	}
	defer closer.ContextCloserWithLogOnError(ctx, "IPFS node", n)

	ipfsClient := n.Client()

	err = func() error {
		log.Ctx(ctx).Debug().
			Str("cid", result.Data.CID).
			Str("name", result.Data.Name).
			Str("path", downloadPath).
			Msg("Downloading result CID")

		innerCtx, cancel := context.WithTimeout(ctx, d.settings.Timeout)
		defer cancel()

		return ipfsClient.Get(innerCtx, result.Data.CID, downloadPath)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result")
		}

		return err
	}
	return nil
}
