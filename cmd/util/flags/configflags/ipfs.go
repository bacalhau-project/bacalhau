package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var IPFSFlags = []Definition{
	{
		FlagName:          "ipfs-connect",
		ConfigPath:        "ipfs.connect.deprecated",
		DefaultValue:      "",
		FailIfUsed:        true,
		Deprecated:        true,
		DeprecatedMessage: "Use one of: ipfs-connect-storage, ipfs-connect-publisher, ipfs-connect-downloader",
	},
	{
		FlagName:             "ipfs-connect-storage",
		ConfigPath:           cfgtypes.InputSourcesIPFSEndpointKey,
		DefaultValue:         cfgtypes.Default.InputSources.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for inputs, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
	{
		FlagName:             "ipfs-connect-publisher",
		ConfigPath:           cfgtypes.PublishersIPFSEndpointKey,
		DefaultValue:         cfgtypes.Default.Publishers.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for publishing, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
	{
		FlagName:             "ipfs-connect-downloader",
		ConfigPath:           cfgtypes.ResultDownloadersIPFSEndpointKey,
		DefaultValue:         cfgtypes.Default.ResultDownloaders.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for downloading, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
}
