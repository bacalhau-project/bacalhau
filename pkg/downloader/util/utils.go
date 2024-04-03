package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/s3signed"
	ipfspkg "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

func NewStandardDownloaders(
	cm *system.CleanupManager, ipfsConnect string) downloader.DownloaderProvider {
	s3PreSignedDownloader := s3signed.NewDownloader(s3signed.DownloaderParams{
		HTTPDownloader: http.NewHTTPDownloader(),
	})

	downloaders := map[string]downloader.Downloader{
		models.StorageSourceS3PreSigned: s3PreSignedDownloader,
		models.StorageSourceURL:         http.NewHTTPDownloader(),
	}

	if ipfsConnect != "" {
		client, err := ipfspkg.NewClientUsingRemoteHandler(context.Background(), ipfsConnect)
		if err == nil {
			ipfsDownloader := ipfs.NewIPFSDownloader(cm, client)
			downloaders[models.StorageSourceIPFS] = ipfsDownloader
		} else {
			log.Warn().Err(err).Msg("Failed to create IPFS client with remote handler. IPFS downloader will not be available.")
		}
	}

	return provider.NewMappedProvider(downloaders)
}
