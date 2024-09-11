package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3signed"
	ipfs_client "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func NewStandardDownloaders(ctx context.Context, cfg types.ResultDownloaders) (downloader.DownloaderProvider, error) {
	providers := make(map[string]downloader.Downloader)

	if cfg.IsNotDisabled(models.StorageSourceS3PreSigned) {
		providers[models.StorageSourceS3PreSigned] = s3signed.NewDownloader(s3signed.DownloaderParams{
			HTTPDownloader: http.NewHTTPDownloader(),
		})
	}

	if cfg.IsNotDisabled(models.StorageSourceURL) {
		providers[models.StorageSourceURL] = http.NewHTTPDownloader()
	}

	if cfg.IsNotDisabled(models.StorageSourceIPFS) {
		if cfg.Types.IPFS.Endpoint != "" {
			ipfsClient, err := ipfs_client.NewClient(ctx, cfg.Types.IPFS.Endpoint)
			if err != nil {
				return nil, err
			}
			providers[models.StorageSourceIPFS] = ipfs.NewIPFSDownloader(ipfsClient)
		}
	}

	return provider.NewMappedProvider(providers), nil
}
