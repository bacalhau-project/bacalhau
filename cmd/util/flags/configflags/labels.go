package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var LabelFlags = []Definition{
	{
		FlagName:     "labels",
		ConfigPath:   "Compute.Labels",
		DefaultValue: types.Default.Compute.Labels,
		//nolint:lll
		Description: `Labels to be associated with the compute node that can be used for node selection and filtering. (e.g. --labels key1=value1,key2=value2)`,
	},
}
