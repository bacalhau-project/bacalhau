package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var ComputeStorageFlags = []Definition{
	{
		FlagName:             "compute-execution-store-type",
		ConfigPath:           types.NodeComputeExecutionStoreType,
		DefaultValue:         Default.Node.Compute.ExecutionStore.Type,
		Description:          "The type of store used by the compute node (BoltDB or InMemory)",
		EnvironmentVariables: []string{"BACALHAU_COMPUTE_STORE_TYPE"},
	},
	{
		FlagName:             "compute-execution-store-path",
		ConfigPath:           types.NodeComputeExecutionStorePath,
		DefaultValue:         Default.Node.Compute.ExecutionStore.Path,
		Description:          "The path used for the compute execution store when using BoltDB",
		EnvironmentVariables: []string{"BACALHAU_COMPUTE_STORE_PATH"},
	},
}
