package configflags

// deprecated
var NodeTypeFlags = []Definition{
	{
		FlagName:          "node-type",
		ConfigPath:        "node.type.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: "Use --orchestrator and/or --compute to set the node type.",
	},
}
