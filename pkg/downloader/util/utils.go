package util

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/downloader/estuary"
	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewDownloadSettings() *model.DownloaderSettings {
	return &model.DownloaderSettings{
		TimeoutSecs: int(downloader.DefaultIPFSTimeout.Seconds()),
		// we leave this blank so the CLI will auto-create a job folder in pwd
		OutputDir:      "",
		IPFSSwarmAddrs: "",
	}
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
