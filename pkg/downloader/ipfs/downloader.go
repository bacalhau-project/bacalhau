package ipfs

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
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

func (ipfsDownloader *Downloader) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (ipfsDownloader *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader/ipfs.Downloader.FetchResult")
	defer span.End()

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	n, err := spinUpIPFSNode(ctx, ipfsDownloader.cm, ipfsDownloader.settings.IPFSSwarmAddrs)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := n.Close(ctx); closeErr != nil {
			log.Ctx(ctx).Error().Err(closeErr).Msg("Failed to close IPFS node")
		}
	}()

	log.Ctx(ctx).Debug().Msg("Connecting client to new IPFS node...")
	ipfsClient := n.Client()

	err = func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			result.Data.Name,
			result.Data.CID, downloadPath,
		)

		innerCtx, cancel := context.WithDeadline(ctx, time.Now().Add(ipfsDownloader.settings.Timeout))
		defer cancel()

		return ipfsClient.Get(innerCtx, result.Data.CID, downloadPath)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

func spinUpIPFSNode(
	ctx context.Context,
	cm *system.CleanupManager,
	ipfsSwarmAddrs string,
) (*ipfs.Node, error) {
	log.Ctx(ctx).Debug().Msg("Spinning up IPFS node...")
	n, err := ipfs.NewNode(ctx, cm, strings.Split(ipfsSwarmAddrs, ","))
	if err != nil {
		return nil, err
	}
	return n, nil
}
