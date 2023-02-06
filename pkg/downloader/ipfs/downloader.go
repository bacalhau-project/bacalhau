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
	settings   *model.DownloaderSettings
	ipfsNode   *ipfs.LiteNode
	ipfsClient ipfs.LiteClient
}

func NewIPFSDownloader(ctx context.Context, settings *model.DownloaderSettings) (*Downloader, error) {
	var peerAddrs []string
	for _, addr := range strings.Split(settings.IPFSSwarmAddrs, ",") {
		peerAdr := strings.TrimSpace(addr)
		if peerAdr != "" {
			peerAddrs = append(peerAddrs, strings.TrimSpace(addr))
		}
	}

	node, err := ipfs.NewLiteNode(ctx, ipfs.LiteNodeParams{
		PeerAddrs: peerAddrs,
	})
	if err != nil {
		return nil, err
	}

	return &Downloader{
		settings:   settings,
		ipfsNode:   node,
		ipfsClient: node.Client(),
	}, nil
}

func (ipfsDownloader *Downloader) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (ipfsDownloader *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloadClient.ipfs.FetchResult")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			result.Data.Name,
			result.Data.CID, downloadPath,
		)

		innerCtx, cancel := context.WithDeadline(ctx, time.Now().Add(ipfsDownloader.settings.Timeout))
		defer cancel()

		return ipfsDownloader.ipfsClient.Get(innerCtx, result.Data.CID, downloadPath)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

func (ipfsDownloader *Downloader) Close() error {
	return ipfsDownloader.ipfsNode.Close()
}
