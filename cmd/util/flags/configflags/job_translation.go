package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var JobTranslationFlags = []Definition{
	{
		FlagName:     "requester-job-translation-enabled",
		ConfigPath:   cfgtypes.FeatureFlagsExecTranslationKey,
		DefaultValue: cfgtypes.Default.FeatureFlags.ExecTranslation,
		Description:  `Whether jobs should be translated at the requester node or not. Default: false`,
	},
}
