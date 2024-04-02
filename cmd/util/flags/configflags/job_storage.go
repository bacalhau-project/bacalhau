package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var RequesterJobStorageFlags = []Definition{
	{
		FlagName:             "requester-job-store-type",
		ConfigPath:           types.NodeRequesterJobStoreType,
		DefaultValue:         Default.Node.Requester.JobStore.Type,
		Description:          "The type of job store used by the requester node (BoltDB or InMemory)",
		EnvironmentVariables: []string{"BACALHAU_JOB_STORE_TYPE"},
	},
	{
		FlagName:             "requester-job-store-path",
		ConfigPath:           types.NodeRequesterJobStorePath,
		DefaultValue:         Default.Node.Requester.JobStore.Path,
		Description:          "The path used for the requester job store store when using BoltDB",
		EnvironmentVariables: []string{"BACALHAU_JOB_STORE_PATH"},
	},
}
