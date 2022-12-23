package downloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

const DefaultIPFSTimeout time.Duration = 5 * time.Minute

type ipfsDownloader struct {
	Settings *DownloadSettings
	client   *ipfs.Client
}

func NewIPFSDownloader(ctx context.Context, cm *system.CleanupManager, settings *DownloadSettings) (*ipfsDownloader, error) {
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

	return &ipfsDownloader{
		Settings: settings,
		client:   ipfsClient,
	}, nil
}

func (ipfsd *ipfsDownloader) GetResultsOutputDir() (string, error) {
	return filepath.Abs(ipfsd.Settings.OutputDir)
}

func (ipfsd *ipfsDownloader) FetchResults(ctx context.Context, shardCIDContext shardCIDContext) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloader.ipfs.FetchResults")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			shardCIDContext.result.Data.Name,
			shardCIDContext.result.Data.CID, shardCIDContext.cidDownloadDir,
		)

		innerCtx, cancel := context.WithDeadline(ctx,
			time.Now().Add(time.Second*time.Duration(ipfsd.Settings.TimeoutSecs)))
		defer cancel()

		return ipfsd.client.Get(innerCtx, shardCIDContext.result.Data.CID, shardCIDContext.cidDownloadDir)
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
