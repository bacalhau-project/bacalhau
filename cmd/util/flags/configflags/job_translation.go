package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var JobTranslationFlags = []Definition{
	{
		FlagName:     "requester-job-translation-enabled",
		ConfigPath:   types2.FeatureFlagsExecTranslationKey,
		DefaultValue: types2.Default.FeatureFlags.ExecTranslation,
		Description:  `Whether jobs should be translated at the requester node or not. Default: false`,
	},
}
