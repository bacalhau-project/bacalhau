package configflags

// deprecated
var RequesterJobStorageFlags = []Definition{
	{
		FlagName:          "requester-job-store-type",
		ConfigPath:        "requester.job.store.type.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "type is no longer configurable. bacalhau uses BoltDB",
	},
	{
		FlagName:          "requester-job-store-path",
		ConfigPath:        "requester.job.store.path.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "path is no longer configurable. location $BACALHAU_DIR/orchestrator/state_boltdb.db",
	},
}
