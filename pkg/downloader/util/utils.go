package util

import (
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewDownloadSettings() *model.DownloaderSettings {
	settings := model.DownloaderSettings{
		Timeout: model.DefaultIPFSTimeout,
		// we leave this blank so the CLI will auto-create a job folder in pwd
		SingleFile:     "",
		OutputDir:      "",
		IPFSSwarmAddrs: "",
	}
	if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
		settings.IPFSSwarmAddrs = os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES")
		// Assume that an explicitly set IPFS_SWARM_ADDRESSES means we are using
		// a private IPFS cluster and that we do not want to connect to the
		// public one as well.
		//
		// NB: might want to expose a specific toggle to users to allow
		// connecting to specific peers in the public network?
		//
		// TODO: if this doesn't help, investigate whether only having the IPFS
		// of the requester node is causing the issue in devstack?
		settings.LocalIPFS = true
	} else {
		settings.IPFSSwarmAddrs = strings.Join(system.Envs[system.GetEnvironment()].IPFSSwarmAddresses, ",")
	}
	return &settings
}

func NewStandardDownloaders(
	cm *system.CleanupManager,
	settings *model.DownloaderSettings) downloader.DownloaderProvider {
	ipfsDownloader := ipfs.NewIPFSDownloader(cm, settings)
	estuaryDownloader := estuary.NewEstuaryDownloader(cm, settings)

	return model.NewMappedProvider(map[model.StorageSourceType]downloader.Downloader{
		model.StorageSourceIPFS:    ipfsDownloader,
		model.StorageSourceEstuary: estuaryDownloader,
	})
}
