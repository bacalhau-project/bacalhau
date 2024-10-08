package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var CapacityFlags = []Definition{
	// deprecated, with message pointing to corresponding --config flag.
	{
		FlagName:             "limit-total-cpu",
		ConfigPath:           types.ComputeAllocatedCapacityCPUKey,
		DefaultValue:         config.Default.Compute.AllocatedCapacity.CPU,
		Description:          `Total CPU core limit to run all jobs (e.g. 500m, 2, 8, 80%, 10%).`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_CAPACITY_TOTALRESOURCELIMITS_CPU"},
		DeprecatedMessage:    makeDeprecationMessage(types.ComputeAllocatedCapacityCPUKey),
	},
	{
		FlagName:             "limit-total-memory",
		ConfigPath:           types.ComputeAllocatedCapacityMemoryKey,
		DefaultValue:         config.Default.Compute.AllocatedCapacity.Memory,
		Description:          `Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb, 80%, 10%).`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_CAPACITY_TOTALRESOURCELIMITS_MEMORY"},
		DeprecatedMessage:    makeDeprecationMessage(types.ComputeAllocatedCapacityMemoryKey),
	},
	{
		FlagName:             "limit-total-gpu",
		ConfigPath:           types.ComputeAllocatedCapacityGPUKey,
		DefaultValue:         config.Default.Compute.AllocatedCapacity.GPU,
		Description:          `Total GPU limit to run all jobs (e.g. 1, 2, 80%, 10%).`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_CAPACITY_TOTALRESOURCELIMITS_GPU"},
		DeprecatedMessage:    makeDeprecationMessage(types.ComputeAllocatedCapacityGPUKey),
	},

	// deprecated, the feature is no longer supported
	{
		FlagName:             "ignore-physical-resource-limits",
		ConfigPath:           "ignore.physical.resource.limits.deprecated",
		Description:          `When set the compute node will ignore is physical resource limits`,
		EnvironmentVariables: []string{"BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "BACALHAU_NODE_COMPUTE_CAPACITY_IGNOREPHYSICALRESOURCELIMITS"},
		DefaultValue:         "",
		Deprecated:           true,
		DeprecatedMessage:    FeatureDeprecatedMessage,
	},
	{
		FlagName:          "limit-job-cpu",
		ConfigPath:        "limit.job.cpu.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:          "limit-job-memory",
		ConfigPath:        "limit.job.memory.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:          "limit-job-disk",
		ConfigPath:        "limit.job.disk.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:          "limit-job-gpu",
		ConfigPath:        "limit.job.gpu.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
}
