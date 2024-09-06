package configflags

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// deprecated
var PublishingFlags = []Definition{
	{
		FlagName:     "default-publisher",
		ConfigPath:   "default.publisher.deprecated",
		DefaultValue: "",
		Deprecated:   true,
		DeprecatedMessage: fmt.Sprintf("Use one or more of the following options, all are accepted %s, %s",
			makeConfigFlagDeprecationCommand(types.JobDefaultsBatchTaskPublisherConfigTypeKey),
			makeConfigFlagDeprecationCommand(types.JobDefaultsOpsTaskPublisherConfigParamsKey),
		),
	},
}
