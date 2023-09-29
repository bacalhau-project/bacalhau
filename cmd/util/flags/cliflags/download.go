package cliflags

import (
	"time"

	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func NewDefaultDownloaderSettings() *DownloaderSettings {
	return &DownloaderSettings{
		Timeout: model.DefaultDownloadTimeout,
		// we leave this blank so the CLI will auto-create a job folder in pwd
		SingleFile: "",
		OutputDir:  "",
	}
}

type DownloaderSettings struct {
	Timeout    time.Duration
	OutputDir  string
	SingleFile string
	Raw        bool
}

func NewDownloadFlags(settings *DownloaderSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Download flags", pflag.ContinueOnError)
	flags.BoolVar(&settings.Raw, "raw",
		settings.Raw, "Download raw result CIDs instead of merging multiple CIDs into a single result")
	flags.DurationVar(&settings.Timeout, "download-timeout-secs",
		settings.Timeout, "Timeout duration for IPFS downloads.")
	flags.StringVar(&settings.OutputDir, "output-dir",
		settings.OutputDir, "Directory to write the output to.")
	return flags
}
