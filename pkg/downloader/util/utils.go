package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3signed"
	ipfs_client "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func NewStandardDownloaders(ctx context.Context, ipfsConnect string) (downloader.DownloaderProvider, error) {
	providers := map[string]downloader.Downloader{
		models.StorageSourceS3PreSigned: s3signed.NewDownloader(s3signed.DownloaderParams{
			HTTPDownloader: http.NewHTTPDownloader(),
		}),
		models.StorageSourceURL: http.NewHTTPDownloader(),
	}
	if ipfsConnect != "" {
		ipfsClient, err := ipfs_client.NewClient(ctx, ipfsConnect)
		if err != nil {
			return nil, err
		}
		providers[models.StorageSourceIPFS] = ipfs.NewIPFSDownloader(ipfsClient)
	}

	return provider.NewMappedProvider(providers), nil
}
