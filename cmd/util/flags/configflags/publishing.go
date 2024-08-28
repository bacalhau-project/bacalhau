package configflags

// deprecated
var PublishingFlags = []Definition{
	{
		FlagName:          "default-publisher",
		ConfigPath:        "default.publisher.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: "Use -c Job.Defaults.<job_type>.Task.Publisher.Type=<publisher_type> to configure a default publisher for a job.",
	},
}
