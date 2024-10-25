package configflags

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var IPFSFlags = []Definition{
	{
		FlagName:     "ipfs-connect",
		ConfigPath:   "ipfs.connect.deprecated",
		DefaultValue: "",
		Deprecated:   true,
		DeprecatedMessage: fmt.Sprintf("Use one or more of the following options, all are accepted %s, %s, %s",
			makeConfigFlagDeprecationCommand(types.InputSourcesTypesIPFSEndpointKey),
			makeConfigFlagDeprecationCommand(types.PublishersTypesIPFSEndpointKey),
			makeConfigFlagDeprecationCommand(types.ResultDownloadersTypesIPFSEndpointKey),
		),
	},
	{
		FlagName:     "ipfs-connect-storage",
		ConfigPath:   types.InputSourcesTypesIPFSEndpointKey,
		DefaultValue: config.Default.InputSources.Types.IPFS.Endpoint,
		Description: "The ipfs host multiaddress to connect to for inputs, " +
			"otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.InputSourcesTypesIPFSEndpointKey),
	},
	{
		FlagName:     "ipfs-connect-publisher",
		ConfigPath:   types.PublishersTypesIPFSEndpointKey,
		DefaultValue: config.Default.Publishers.Types.IPFS.Endpoint,
		Description: "The ipfs host multiaddress to connect to for publishing, " +
			"otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.PublishersTypesIPFSEndpointKey),
	},
	{
		FlagName:     "ipfs-connect-downloader",
		ConfigPath:   types.ResultDownloadersTypesIPFSEndpointKey,
		DefaultValue: config.Default.ResultDownloaders.Types.IPFS.Endpoint,
		Description: "The ipfs host multiaddress to connect to for downloading, " +
			"otherwise an in-process IPFS node will be created if not set.",
		EnvironmentVariables: []string{"BACALHAU_NODE_IPFS_CONNECT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.ResultDownloadersTypesIPFSEndpointKey),
	},
}
