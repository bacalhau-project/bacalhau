package downloader

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/downloader/estuary"
	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type MappedDownloaderProvider struct {
	downloaders map[model.StorageSourceType]Downloader
}

func NewMappedDownloaderProvider(downloaders map[model.StorageSourceType]Downloader) *MappedDownloaderProvider {
	return &MappedDownloaderProvider{
		downloaders: downloaders,
	}
}

func (p *MappedDownloaderProvider) GetDownloader(storageType model.StorageSourceType) (Downloader, error) {
	downloader, ok := p.downloaders[storageType]
	if !ok {
		return nil, fmt.Errorf(
			"no matching downloader found on this server: %s", storageType)
	}

	return downloader, nil
}

func NewStandardDownloaders(
	cm *system.CleanupManager,
	settings *model.DownloaderSettings) DownloaderProvider {
	ipfsDownloader := ipfs.NewIPFSDownloader(cm, settings)
	estuaryDownloader := estuary.NewEstuaryDownloader(cm, settings)

	return NewMappedDownloaderProvider(map[model.StorageSourceType]Downloader{
		model.StorageSourceIPFS:    ipfsDownloader,
		model.StorageSourceEstuary: estuaryDownloader,
	})
}
