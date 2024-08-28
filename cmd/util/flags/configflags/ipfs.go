package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var IPFSFlags = []Definition{
	{
		FlagName:          "ipfs-connect",
		ConfigPath:        "ipfs.connect.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: "Use one of: ipfs-connect-storage, ipfs-connect-publisher, ipfs-connect-downloader",
	},
	{
		FlagName:             "ipfs-connect-storage",
		ConfigPath:           "InputSources.IPFS.Endpoint",
		DefaultValue:         types2.Default.InputSources.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for inputs, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
	{
		FlagName:             "ipfs-connect-publisher",
		ConfigPath:           "Publisher.IPFS.Endpoint",
		DefaultValue:         types2.Default.Publishers.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for publishing, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
	{
		FlagName:             "ipfs-connect-downloader",
		ConfigPath:           "ResultDownloaders.IPFS.Endpoint",
		DefaultValue:         types2.Default.ResultDownloaders.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for downloading, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
	},
}
