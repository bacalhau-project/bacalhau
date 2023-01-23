package util

import (
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/downloader/estuary"
	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

func NewDownloadSettings() *model.DownloaderSettings {
	settings := model.DownloaderSettings{
		Timeout: model.DefaultIPFSTimeout,
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

func NewStandardDownloaders(
	cm *system.CleanupManager,
	settings *model.DownloaderSettings) downloader.DownloaderProvider {
	ipfsDownloader := ipfs.NewIPFSDownloader(cm, settings)
	estuaryDownloader := estuary.NewEstuaryDownloader(cm, settings)

	return downloader.NewMappedDownloaderProvider(map[model.StorageSourceType]downloader.Downloader{
		model.StorageSourceIPFS:    ipfsDownloader,
		model.StorageSourceEstuary: estuaryDownloader,
	})
}
