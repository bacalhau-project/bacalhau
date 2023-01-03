package util

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/downloader/estuary"
	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
)

func NewDownloadSettings() *model.DownloaderSettings {
	settings := model.DownloaderSettings{
		TimeoutSecs: downloader.DefaultIPFSTimeout,
		// we leave this blank so the CLI will auto-create a job folder in pwd
		OutputDir:      "",
		IPFSSwarmAddrs: "",
	}

	switch system.GetEnvironment() {
	case system.EnvironmentProd:
		settings.IPFSSwarmAddrs = strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ",")
	case system.EnvironmentTest:
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			log.Warn().Msg("No action (don't use BACALHAU_IPFS_SWARM_ADDRESSES")
		}
	case system.EnvironmentDev:
		// TODO: add more dev swarm addresses?
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			settings.IPFSSwarmAddrs = os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES")
		}
	case system.EnvironmentStaging:
		log.Warn().Msg("Staging environment has no IPFS swarm addresses attached")
	}

	return &settings
}

func NewIPFSDownloaders(
	ctx context.Context,
	cm *system.CleanupManager,
	settings *model.DownloaderSettings) (downloader.DownloaderProvider, error) {
	ipfsDownloader, err := ipfs.NewIPFSDownloader(ctx, cm, settings)
	if err != nil {
		return nil, err
	}

	estuaryDownloader, err := estuary.NewEstuaryDownloader(settings)
	if err != nil {
		return nil, err
	}

	return downloader.NewMappedDownloaderProvider(map[model.StorageSourceType]downloader.Downloader{
		model.StorageSourceIPFS:    ipfsDownloader,
		model.StorageSourceEstuary: estuaryDownloader,
	}), nil
}
