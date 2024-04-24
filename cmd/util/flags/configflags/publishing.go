package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var PublishingFlags = []Definition{
	{
		FlagName:     "default-publisher",
		DefaultValue: Default.Node.Requester.JobDefaults.DefaultPublisher,
		ConfigPath:   types.NodeRequesterJobDefaultsDefaultPublisher,
		Description:  `A default publisher to apply to all jobs without a publisher`,
	},
}
