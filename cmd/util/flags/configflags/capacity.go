package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var CapacityFlags = []Definition{
	{
		FlagName:     "capacity-cpu",
		ConfigPath:   "Compute.AllocatedCapacity.CPU",
		DefaultValue: types2.Default.Compute.AllocatedCapacity.CPU,
		Description:  `Total CPU core limit to run all jobs (e.g. 500m, 2, 8, 80%, 10%).`,
	},
	{
		FlagName:     "capacity-memory",
		ConfigPath:   "Compute.AllocatedCapacity.Memory",
		DefaultValue: types2.Default.Compute.AllocatedCapacity.Memory,
		Description:  `Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb, 80%, 10%).`,
	},
	{
		FlagName:     "capacity-disk",
		ConfigPath:   "Compute.AllocatedCapacity.Disk",
		DefaultValue: types2.Default.Compute.AllocatedCapacity.Disk,
		Description:  `Total Disk limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb, 80%, 10%).`,
	},
	{
		FlagName:     "capacity-gpu",
		ConfigPath:   "Compute.AllocatedCapacity.GPU",
		DefaultValue: types2.Default.Compute.AllocatedCapacity.GPU,
		Description:  `Total GPU limit to run all jobs (e.g. 1, 2, 80%, 10%).`,
	},
}
