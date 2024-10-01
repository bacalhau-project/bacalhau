package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var JobTranslationFlags = []Definition{
	{
		FlagName:             "requester-job-translation-enabled",
		ConfigPath:           types.FeatureFlagsExecTranslationKey,
		DefaultValue:         types.Default.FeatureFlags.ExecTranslation,
		Description:          `Whether jobs should be translated at the requester node or not. Default: false`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_REQUESTER_TRANSLATIONENABLED"},
		DeprecatedMessage:    makeDeprecationMessage(types.FeatureFlagsExecTranslationKey),
	},
}
