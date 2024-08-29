package configflags

// deprecated
var ComputeStorageFlags = []Definition{
	{
		FlagName:          "compute-execution-store-type",
		ConfigPath:        "compute.execution.store.type.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "type is no longer configurable. bacalhau uses BoltDB",
	},
	{
		FlagName:          "compute-execution-store-path",
		ConfigPath:        "compute.execution.store.path.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		FailIfUsed:        true,
		DeprecatedMessage: "path is no longer configurable. location $BACALHAU_DIR/compute/state_boltdb.db",
	},
}
