package util

import (
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewStandardDownloaders(
	cm *system.CleanupManager,
	settings *model.DownloaderSettings) downloader.DownloaderProvider {
	ipfsDownloader := ipfs.NewIPFSDownloader(cm, settings)
	estuaryDownloader := estuary.NewEstuaryDownloader(cm, settings)

	return provider.NewMappedProvider(map[string]downloader.Downloader{
		model.StorageSourceIPFS.String():    ipfsDownloader,
		model.StorageSourceEstuary.String(): estuaryDownloader,
	})
}
