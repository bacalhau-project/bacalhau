package util

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3signed"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewStandardDownloaders(cm *system.CleanupManager) (downloader.DownloaderProvider, error) {
	ipfsDownloader := ipfs.NewIPFSDownloader(cm)
	s3PreSignedDownloader := s3signed.NewDownloader(s3signed.DownloaderParams{
		HTTPDownloader: http.NewHTTPDownloader(),
	})

	cfg, err := s3helper.DefaultAWSConfig()
	if err != nil {
		return nil, fmt.Errorf("getting default aws config: %w", err)
	}
	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})
	s3Downloader := s3.NewDownloader(clientProvider)

	return provider.NewMappedProvider(map[string]downloader.Downloader{
		models.StorageSourceIPFS:        ipfsDownloader,
		models.StorageSourceS3PreSigned: s3PreSignedDownloader,
		models.StorageSourceS3:          s3Downloader,
	}), nil
}
