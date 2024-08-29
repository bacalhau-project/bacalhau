package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var CapacityFlags = []Definition{
	{
		FlagName:     "limit-total-cpu",
		ConfigPath:   types2.ComputeAllocatedCapacityCPUKey,
		DefaultValue: types2.Default.Compute.AllocatedCapacity.CPU,
		Description:  `Total CPU core limit to run all jobs (e.g. 500m, 2, 8, 80%, 10%).`,
	},
	{
		FlagName:     "limit-total-memory",
		ConfigPath:   types2.ComputeAllocatedCapacityMemoryKey,
		DefaultValue: types2.Default.Compute.AllocatedCapacity.Memory,
		Description:  `Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb, 80%, 10%).`,
	},
	{
		FlagName:     "limit-total-disk",
		ConfigPath:   types2.ComputeAllocatedCapacityDiskKey,
		DefaultValue: types2.Default.Compute.AllocatedCapacity.Disk,
		Description:  `Total Disk limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb, 80%, 10%).`,
	},
	{
		FlagName:     "limit-total-gpu",
		ConfigPath:   types2.ComputeAllocatedCapacityGPUKey,
		DefaultValue: types2.Default.Compute.AllocatedCapacity.GPU,
		Description:  `Total GPU limit to run all jobs (e.g. 1, 2, 80%, 10%).`,
	},

	// deprecated
	{
		FlagName:          "limit-job-cpu",
		ConfigPath:        "limit.job.cpu.deprecated",
		DefaultValue:      "",
		FailIfUsed:        true,
		Deprecated:        true,
		DeprecatedMessage: "use limit-total-cpu.",
	},
	{
		FlagName:          "limit-job-memory",
		ConfigPath:        "limit.job.memory.deprecated",
		DefaultValue:      "",
		FailIfUsed:        true,
		Deprecated:        true,
		DeprecatedMessage: "use limit-total-memory.",
	},
	{
		FlagName:          "limit-job-disk",
		ConfigPath:        "limit.job.disk.deprecated",
		DefaultValue:      "",
		FailIfUsed:        true,
		Deprecated:        true,
		DeprecatedMessage: "use limit-total-disk.",
	},
	{
		FlagName:          "limit-job-gpu",
		ConfigPath:        "limit.job.gpu.deprecated",
		DefaultValue:      "",
		FailIfUsed:        true,
		Deprecated:        true,
		DeprecatedMessage: "use limit-total-gpu.",
	},
}
