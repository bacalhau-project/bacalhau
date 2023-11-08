package util

import (
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/iroh"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewStandardDownloaders(
	cm *system.CleanupManager,
	settings *model.DownloaderSettings) downloader.DownloaderProvider {
	//ipfsDownloader := ipfs.NewIPFSDownloader(cm, settings)

	client, err := iroh.New("/Users/frrist/Workspace/src/github.com/bacalhau-project/bacalhau/irohrepo")
	if err != nil {
		panic(err)
	}
	return provider.NewSingletonProvider[downloader.Downloader](model.StorageSourceIroh.String(), client)
}
