package ipfs

import (
	"context"
	"errors"
	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type IPFSDownloader struct {
	Settings *downloader.DownloadSettings
	Client   *ipfs.Client
}

func NewIPFSDownloader(ctx context.Context, cm *system.CleanupManager, settings *downloader.DownloadSettings) (*IPFSDownloader, error) {
	switch system.GetEnvironment() {
	case system.EnvironmentProd:
		settings.IPFSSwarmAddrs = strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ",")
	case system.EnvironmentTest:
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			log.Ctx(ctx).Warn().Msg("No action (don't use BACALHAU_IPFS_SWARM_ADDRESSES")
		}
	case system.EnvironmentDev:
		// TODO: add more dev swarm addresses?
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			settings.IPFSSwarmAddrs = os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES")
		}
	case system.EnvironmentStaging:
		log.Ctx(ctx).Warn().Msg("Staging environment has no IPFS swarm addresses attached")
	}

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	n, err := spinUpIPFSNode(ctx, cm, settings.IPFSSwarmAddrs)
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Debug().Msg("Connecting client to new IPFS node...")
	ipfsClient, err := n.Client()
	if err != nil {
		return nil, err
	}

	return &IPFSDownloader{
		Settings: settings,
		Client:   ipfsClient,
	}, nil
}

func (ipfsDownloader *IPFSDownloader) GetResultsOutputDir() (string, error) {
	return filepath.Abs(ipfsDownloader.Settings.OutputDir)
}

func (ipfsDownloader *IPFSDownloader) FetchResult(ctx context.Context, shardCIDContext downloader.ShardCIDContext) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloadClient.ipfs.FetchResult")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			shardCIDContext.Result.Data.Name,
			shardCIDContext.Result.Data.CID, shardCIDContext.CIDDownloadDir,
		)

		innerCtx, cancel := context.WithDeadline(ctx,
			time.Now().Add(time.Second*time.Duration(ipfsDownloader.Settings.TimeoutSecs)))
		defer cancel()

		return ipfsDownloader.Client.Get(innerCtx, shardCIDContext.Result.Data.CID, shardCIDContext.CIDDownloadDir)
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
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.DownloadJob.SpinningUpIPFS")
	defer span.End()

	log.Ctx(ctx).Debug().Msg("Spinning up IPFS node...")
	n, err := ipfs.NewNode(ctx, cm, strings.Split(ipfsSwarmAddrs, ","))
	if err != nil {
		return nil, err
	}
	return n, nil
}
