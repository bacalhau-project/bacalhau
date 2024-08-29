package configflags

// deprecated
var ComputeTimeoutFlags = []Definition{
	{
		FlagName:          "job-execution-timeout-bypass-client-id",
		ConfigPath:        "job.execution.timeout.bypass.client.id.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "feature is deprecated",
	},
	{
		FlagName:          "job-negotiation-timeout",
		ConfigPath:        "job.negotiation.timeout.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "configuration option is deprecated",
	},
	{
		FlagName:          "min-job-execution-timeout",
		ConfigPath:        "min.job.execution.timeout.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "configuration option is deprecated",
	},
	{
		FlagName:          "max-job-execution-timeout",
		ConfigPath:        "max.job.execution.timeout.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "configuration option is deprecated",
	},
	{
		FlagName:          "default-job-execution-timeout",
		ConfigPath:        "default.job.execution.timeout.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "Use -c Job.Defaults.<Batch|Ops>.Task.Timeouts.TotalTimeout=<duration> to configure a default execution timeout for a job.",
	},
}
