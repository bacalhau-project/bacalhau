package util

import (
	"context"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/downloader/estuary"
	"github.com/filecoin-project/bacalhau/pkg/downloader/http"
	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewDownloadSettings() *model.DownloaderSettings {
	settings := model.DownloaderSettings{
		Timeout: model.DefaultIPFSTimeout,
		// we leave this blank so the CLI will auto-create a job folder in pwd
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
	ctx context.Context,
	cm *system.CleanupManager,
	settings *model.DownloaderSettings) (downloader.DownloaderProvider, error) {
	ipfsDownloader, err := ipfs.NewIPFSDownloader(ctx, settings)
	if err != nil {
		return nil, err
	}
	httpDownloader := http.NewHTTPDownloader(settings)
	estuaryDownloader := estuary.NewEstuaryDownloader(estuary.DownloaderParams{
		IPFSDownloader: ipfsDownloader,
		HTTPDownloader: httpDownloader,
	})

	cm.RegisterCallback(ipfsDownloader.Close)

	return model.NewMappedProvider(map[model.StorageSourceType]downloader.Downloader{
		model.StorageSourceIPFS:    ipfsDownloader,
		model.StorageSourceEstuary: estuaryDownloader,
	}), nil
}
