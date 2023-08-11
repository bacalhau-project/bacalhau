package util

import (
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system/cleanup"
)

func NewStandardDownloaders(
	cm *cleanup.CleanupManager,
	settings *model.DownloaderSettings) downloader.DownloaderProvider {
	ipfsDownloader := ipfs.NewIPFSDownloader(cm, settings)
	estuaryDownloader := estuary.NewEstuaryDownloader(cm, settings)

	return model.NewMappedProvider(map[model.StorageSourceType]downloader.Downloader{
		model.StorageSourceIPFS:    ipfsDownloader,
		model.StorageSourceEstuary: estuaryDownloader,
	})
}
