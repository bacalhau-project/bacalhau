package flags

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewDefaultDownloaderSettings() *DownloaderSettings {
	settings := &DownloaderSettings{
		Timeout: v1beta2.DefaultIPFSTimeout,
		// we leave this blank so the CLI will auto-create a job folder in pwd
		SingleFile:     "",
		OutputDir:      "",
		IPFSSwarmAddrs: "",
	}
	if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
		settings.IPFSSwarmAddrs = os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES")
	} else {
		settings.IPFSSwarmAddrs = strings.Join(system.Envs[system.GetEnvironment()].IPFSSwarmAddresses, ",")
	}
	return settings
}

type DownloaderSettings struct {
	Timeout        time.Duration
	OutputDir      string
	IPFSSwarmAddrs string
	SingleFile     string
	LocalIPFS      bool
	Raw            bool
}

func NewDownloadFlags(settings *DownloaderSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("IPFS Download flags", pflag.ContinueOnError)
	flags.BoolVar(&settings.Raw, "raw",
		settings.Raw, "Download raw result CIDs instead of merging multiple CIDs into a single result")
	flags.DurationVar(&settings.Timeout, "download-timeout-secs",
		settings.Timeout, "Timeout duration for IPFS downloads.")
	flags.StringVar(&settings.OutputDir, "output-dir",
		settings.OutputDir, "Directory to write the output to.")
	flags.StringVar(&settings.IPFSSwarmAddrs, "ipfs-swarm-addrs",
		settings.IPFSSwarmAddrs, "Comma-separated list of IPFS nodes to connect to.")
	return flags
}
