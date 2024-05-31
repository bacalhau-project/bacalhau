package util

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3signed"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewStandardDownloaders(cm *system.CleanupManager, cfg types.IpfsConfig) downloader.DownloaderProvider {
	ipfsDownloader := ipfs.NewIPFSDownloader(cm, cfg)
	s3PreSignedDownloader := s3signed.NewDownloader(s3signed.DownloaderParams{
		HTTPDownloader: http.NewHTTPDownloader(),
	})

	return provider.NewMappedProvider(map[string]downloader.Downloader{
		models.StorageSourceIPFS:        ipfsDownloader,
		models.StorageSourceS3PreSigned: s3PreSignedDownloader,
		models.StorageSourceURL:         http.NewHTTPDownloader(),
	})
}
