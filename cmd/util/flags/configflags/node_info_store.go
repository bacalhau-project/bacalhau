package configflags

// deprecated
var NodeInfoStoreFlags = []Definition{
	{
		FlagName:          "node-info-store-ttl",
		ConfigPath:        "node.info.store.ttl.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
}
