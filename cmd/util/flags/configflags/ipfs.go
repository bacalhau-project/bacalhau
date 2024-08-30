package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
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
		ConfigPath:           types.InputSourcesIPFSEndpointKey,
		DefaultValue:         types.Default.InputSources.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for inputs, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
	{
		FlagName:             "ipfs-connect-publisher",
		ConfigPath:           types.PublishersIPFSEndpointKey,
		DefaultValue:         types.Default.Publishers.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for publishing, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
	{
		FlagName:             "ipfs-connect-downloader",
		ConfigPath:           types.ResultDownloadersIPFSEndpointKey,
		DefaultValue:         types.Default.ResultDownloaders.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for downloading, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
}
