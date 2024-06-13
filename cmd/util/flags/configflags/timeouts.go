package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var ComputeTimeoutFlags = []Definition{
	{
		FlagName:     "job-execution-timeout-bypass-client-id",
		ConfigPath:   types.NodeComputeJobTimeoutsJobExecutionTimeoutClientIDBypassList,
		DefaultValue: Default.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList,
		Description:  `List of IDs of clients that are allowed to bypass the job execution timeout check`,
	},
	{
		FlagName:     "job-negotiation-timeout",
		ConfigPath:   types.NodeComputeJobTimeoutsJobNegotiationTimeout,
		DefaultValue: Default.Node.Compute.JobTimeouts.JobNegotiationTimeout,
		Description:  `Timeout value to hold a bid for a job.`,
	},
	{
		FlagName:     "min-job-execution-timeout",
		ConfigPath:   types.NodeComputeJobTimeoutsMinJobExecutionTimeout,
		DefaultValue: Default.Node.Compute.JobTimeouts.MinJobExecutionTimeout,
		Description:  `The minimum execution timeout this compute node supports. Jobs with lower timeout requirements will not be bid on.`,
	},
	{
		FlagName:     "max-job-execution-timeout",
		ConfigPath:   types.NodeComputeJobTimeoutsMaxJobExecutionTimeout,
		DefaultValue: Default.Node.Compute.JobTimeouts.MaxJobExecutionTimeout,
		Description:  `The maximum execution timeout this compute node supports. Jobs with higher timeout requirements will not be bid on.`,
	},
	{
		FlagName:     "default-job-execution-timeout",
		ConfigPath:   types.NodeComputeJobTimeoutsDefaultJobExecutionTimeout,
		DefaultValue: Default.Node.Compute.JobTimeouts.DefaultJobExecutionTimeout,
		Description:  `default value for the execution timeout this compute node will assign to jobs with no timeout requirement defined.`,
	},
}
