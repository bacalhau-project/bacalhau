package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var CapacityFlags = []Definition{
	{
		FlagName:             "ignore-physical-resource-limits",
		ConfigPath:           types.NodeComputeCapacityIgnorePhysicalResourceLimits,
		DefaultValue:         Default.Node.Compute.Capacity.IgnorePhysicalResourceLimits,
		Description:          `When set the compute node will ignore is physical resource limits`,
		EnvironmentVariables: []string{"BACALHAU_CAPACITY_MANAGER_OVER_COMMIT"},
	},
	{
		FlagName:     "limit-total-cpu",
		ConfigPath:   types.NodeComputeCapacityTotalResourceLimitsCPU,
		DefaultValue: Default.Node.Compute.Capacity.TotalResourceLimits.CPU,
		Description:  `Total CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	},
	{
		FlagName:     "limit-total-memory",
		ConfigPath:   types.NodeComputeCapacityTotalResourceLimitsMemory,
		DefaultValue: Default.Node.Compute.Capacity.TotalResourceLimits.Memory,
		Description:  `Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	},
	{
		FlagName:     "limit-total-gpu",
		ConfigPath:   types.NodeComputeCapacityTotalResourceLimitsGPU,
		DefaultValue: Default.Node.Compute.Capacity.TotalResourceLimits.GPU,
		Description:  `Total GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	},
	{
		FlagName:     "limit-job-cpu",
		ConfigPath:   types.NodeComputeCapacityJobResourceLimitsCPU,
		DefaultValue: Default.Node.Compute.Capacity.JobResourceLimits.CPU,
		Description:  `Job CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	},
	{
		FlagName:     "limit-job-memory",
		ConfigPath:   types.NodeComputeCapacityDefaultJobResourceLimitsMemory,
		DefaultValue: Default.Node.Compute.Capacity.JobResourceLimits.Memory,
		Description:  `Job Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	},
	{
		FlagName:     "limit-job-gpu",
		ConfigPath:   types.NodeComputeCapacityJobResourceLimitsGPU,
		DefaultValue: Default.Node.Compute.Capacity.JobResourceLimits.GPU,
		Description:  `Job GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	},
}
