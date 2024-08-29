package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3signed"
	ipfs_client "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func NewStandardDownloaders(ctx context.Context, cfg cfgtypes.ResultDownloaders) (downloader.DownloaderProvider, error) {
	providers := make(map[string]downloader.Downloader)

	if cfg.Enabled(models.StorageSourceS3PreSigned) {
		providers[models.StorageSourceS3PreSigned] = s3signed.NewDownloader(s3signed.DownloaderParams{
			HTTPDownloader: http.NewHTTPDownloader(),
		})
	}

	if cfg.Enabled(models.StorageSourceURL) {
		providers[models.StorageSourceURL] = http.NewHTTPDownloader()
	}

	if cfg.Enabled(models.StorageSourceIPFS) {
		if cfg.IPFS.Installed() {
			ipfsClient, err := ipfs_client.NewClient(ctx, cfg.IPFS.Endpoint)
			if err != nil {
				return nil, err
			}
			providers[models.StorageSourceIPFS] = ipfs.NewIPFSDownloader(ipfsClient)
		}
	}

	return provider.NewMappedProvider(providers), nil
}
