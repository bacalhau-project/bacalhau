package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var LabelFlags = []Definition{
	{
		FlagName:     "labels",
		ConfigPath:   types.LabelsKey,
		DefaultValue: config.Default.Labels,
		//nolint:lll
		Description:          `Labels to be associated with the node that can be used for node selection and filtering. (e.g. --labels key1=value1,key2=value2)`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_LABELS"},
		DeprecatedMessage:    makeDeprecationMessage(types.LabelsKey),
	},
}
