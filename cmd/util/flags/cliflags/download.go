package cliflags

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
)

func DefaultDownloaderSettings() *DownloaderSettings {
	return &DownloaderSettings{
		Timeout: downloader.DefaultDownloadTimeout,
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

func RegisterDownloadFlags(cmd *cobra.Command, s *DownloaderSettings) {
	fs := pflag.NewFlagSet("Download flags", pflag.ContinueOnError)
	fs.BoolVar(&s.Raw, "raw", s.Raw,
		"Download raw result CIDs instead of merging multiple CIDs into a single result")
	fs.DurationVar(&s.Timeout, "download-timeout-secs", s.Timeout,
		"Timeout duration for IPFS downloads.")
	fs.StringVar(&s.OutputDir, "output-dir", s.OutputDir,
		"Directory to write the output to.")

	cmd.Flags().AddFlagSet(fs)
}
