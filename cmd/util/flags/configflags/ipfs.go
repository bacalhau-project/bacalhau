package configflags

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var IPFSFlags = []Definition{
	{
		FlagName:     "ipfs-connect",
		ConfigPath:   "ipfs.connect.deprecated",
		DefaultValue: "",
		Deprecated:   true,
		DeprecatedMessage: fmt.Sprintf("Use one or more of the following options, all are accepted %s, %s, %s",
			makeConfigFlagDeprecationCommand(types.InputSourcesIPFSEndpointKey),
			makeConfigFlagDeprecationCommand(types.PublishersIPFSEndpointKey),
			makeConfigFlagDeprecationCommand(types.ResultDownloadersIPFSEndpointKey),
		),
	},
	{
		FlagName:             "ipfs-connect-storage",
		ConfigPath:           types.InputSourcesIPFSEndpointKey,
		DefaultValue:         types.Default.InputSources.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for inputs, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.InputSourcesIPFSEndpointKey),
	},
	{
		FlagName:             "ipfs-connect-publisher",
		ConfigPath:           types.PublishersIPFSEndpointKey,
		DefaultValue:         types.Default.Publishers.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for publishing, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.PublishersIPFSEndpointKey),
	},
	{
		FlagName:             "ipfs-connect-downloader",
		ConfigPath:           types.ResultDownloadersIPFSEndpointKey,
		DefaultValue:         types.Default.ResultDownloaders.IPFS.Endpoint,
		Description:          "The ipfs host multiaddress to connect to for downloading, otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.ResultDownloadersIPFSEndpointKey),
	},
}
