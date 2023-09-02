package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var NodeTypeFlags = []Definition{
	{
		FlagName:     "node-type",
		ConfigPath:   types.NodeType,
		DefaultValue: Default.Node.Type,
		Description:  `Whether the node is a compute, requester or both.`,
	},
}
