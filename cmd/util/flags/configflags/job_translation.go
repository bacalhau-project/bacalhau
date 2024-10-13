package configflags

import (
	legacy_types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
)

var JobTranslationFlags = []Definition{
	{
		FlagName:          "requester-job-translation-enabled",
		ConfigPath:        legacy_types.NodeRequesterTranslationEnabled,
		DefaultValue:      false,
		Deprecated:        true,
		DeprecatedMessage: "job translation was an experimental feature and is no longer supported",
	},
}
