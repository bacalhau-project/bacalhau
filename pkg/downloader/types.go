package downloader

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
	DownloadRawFolderName    = "raw"
	DownloadFolderPerm       = 0755
	DownloadFilePerm         = 0644
	DefaultDownloadTimeout   = 5 * time.Minute
)

type Downloader interface {
	provider.Providable

	// FetchResult fetches item and saves to disk (as per item's Target)
	FetchResult(ctx context.Context, item DownloadItem) (string, error)
}

type DownloaderProvider interface {
	provider.Provider[Downloader]
}

type DownloaderSettings struct {
	Timeout    time.Duration
	OutputDir  string
	SingleFile string
	Raw        bool
}

type DownloadItem struct {
	Result     *models.SpecConfig
	SingleFile string
	ParentPath string
}
