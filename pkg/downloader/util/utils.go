package util

import (
	"os"
	"strings"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	spec_estuary "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/estuary"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
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

	return model.NewMappedProvider(map[cid.Cid]downloader.Downloader{
		spec_ipfs.StorageType:    ipfsDownloader,
		spec_estuary.StorageType: estuaryDownloader,
	})
}
