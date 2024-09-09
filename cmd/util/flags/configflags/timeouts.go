package configflags

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// deprecated
var ComputeTimeoutFlags = []Definition{
	{
		FlagName:          "job-execution-timeout-bypass-client-id",
		ConfigPath:        "job.execution.timeout.bypass.client.id.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:          "job-negotiation-timeout",
		ConfigPath:        "job.negotiation.timeout.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:          "min-job-execution-timeout",
		ConfigPath:        "min.job.execution.timeout.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:          "max-job-execution-timeout",
		ConfigPath:        "max.job.execution.timeout.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:     "default-job-execution-timeout",
		ConfigPath:   "default.job.execution.timeout.deprecated",
		DefaultValue: "",
		Deprecated:   true,
		DeprecatedMessage: fmt.Sprintf("Use one or more of the following options, all are accepted %s, %s",
			makeConfigFlagDeprecationCommand(types.JobDefaultsBatchTaskTimeoutsExecutionTimeoutKey),
			makeConfigFlagDeprecationCommand(types.JobDefaultsOpsTaskTimeoutsExecutionTimeoutKey),
		),
	},
}
