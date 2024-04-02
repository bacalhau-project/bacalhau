package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var JobTranslationFlags = []Definition{
	{
		FlagName:     "requester-job-translation-enabled",
		DefaultValue: Default.Node.Requester.TranslationEnabled,
		ConfigPath:   types.NodeRequesterTranslationEnabled,
		Description:  `Whether jobs should be translated at the requester node or not. Default: false`,
	},
}
